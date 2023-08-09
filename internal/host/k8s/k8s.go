// © 2023 SolarWinds Worldwide, LLC. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package k8s

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	collector "github.com/solarwindscloud/apm-proto/go/collectorpb"
	"github.com/solarwindscloud/solarwinds-apm-go/internal/log"
	"github.com/solarwindscloud/solarwinds-apm-go/internal/swotel/semconv"
	"go.opentelemetry.io/otel/sdk/resource"
	"os"
	"regexp"
	"runtime"
	"strings"
	"sync"
)

const (
	linuxNamespaceFile   = "/run/secrets/kubernetes.io/serviceaccount/namespace"
	linuxProcMountInfo   = "/proc/self/mountinfo"
	windowsNamespaceFile = `C:\var\run\secrets\kubernetes.io\serviceaccount\namespace`
)

var uuidRegex = regexp.MustCompile("[[:xdigit:]]{8}-([[:xdigit:]]{4}-){3}[[:xdigit:]]{12}")

var (
	memoized *Metadata
	once     sync.Once
)

type Metadata struct {
	Namespace string
	PodName   string
	PodUid    string
}

func (m *Metadata) ToPB() *collector.K8S {
	return &collector.K8S{
		Namespace: m.Namespace,
		PodName:   m.PodName,
		PodUid:    m.PodUid,
	}
}

func determineNamspaceFileForOS() string {
	//goland:noinspection GoBoolExpressions
	if runtime.GOOS == "windows" {
		return windowsNamespaceFile
	}
	return linuxNamespaceFile
}

func MemoizeMetadata() *Metadata {
	once.Do(func() {
		var err error
		memoized, err = requestMetadata()
		if err != nil {
			log.Debugf("error when retrieving k8s metadata %s", err)
		} else {
			log.Debugf("retrieved k8s metadata: %+v", memoized)
		}
	})
	return memoized
}

func requestMetadata() (*Metadata, error) {
	var namespace, podName, podUid string
	var err error

	// `k8s.*` attributes in `OTEL_RESOURCE_ATTRIBUTES` take precedence
	if otelRes, err := resource.New(context.Background(), resource.WithFromEnv()); err != nil {
		log.Debugf("otel resource detector failed: %s; continuing", err)
	} else {
		kvs := otelRes.Set()
		if val, ok := kvs.Value(semconv.K8SNamespaceNameKey); ok {
			namespace = val.AsString()
		}
		if val, ok := kvs.Value(semconv.K8SPodNameKey); ok {
			podName = val.AsString()
		}
		if val, ok := kvs.Value(semconv.K8SPodUIDKey); ok {
			podUid = val.AsString()
		}
	}

	// Now, check `SW_K8S_*` environment variables
	if namespace == "" {
		namespace = os.Getenv("SW_K8S_POD_NAMESPACE")
	}
	if podName == "" {
		podName = os.Getenv("SW_K8S_POD_NAME")
	}
	if podUid == "" {
		podUid = os.Getenv("SW_K8S_POD_UID")
	}

	// Now fallbacks

	if namespace == "" {
		// If we don't find a namespace, we skip the rest
		namespace, err = getNamespaceFromFile(determineNamspaceFileForOS())
		if err != nil {
			return nil, err
		} else if namespace == "" {
			return nil, errors.New("k8s namespace was empty")
		}
	}

	if podName == "" {
		podName, err = getPodnameFromHostname()
		if err != nil {
			log.Debugf("could not retrieve k8s podname %s, continuing", err)
		}
	}

	if podUid == "" {
		// This function will only fallback when GOOS == "linux", so we always pass in `linuxProcMountInfo` as the filename
		podUid, err = getPodUidFromFile(linuxProcMountInfo)
		if err != nil {
			log.Debugf("could not retrieve k8s podUid %s, continuing", err)
		}
	}

	return &Metadata{
		Namespace: namespace,
		PodName:   podName,
		PodUid:    podUid,
	}, nil
}

func getNamespaceFromFile(fn string) (string, error) {
	log.Debugf("Attempting to read namespace from %s", fn)
	if ns, err := os.ReadFile(fn); err != nil {
		return "", err
	} else {
		return string(ns), nil
	}
}

func getPodnameFromHostname() (string, error) {
	log.Debugf("Returning hostname as k8s pod name")
	return os.Hostname()
}

func getPodUidFromFile(fn string) (string, error) {
	//goland:noinspection GoBoolExpressions
	if runtime.GOOS == "linux" {
		return getPodUidFromProc(fn)
	} else {
		log.Debugf("Cannot determine k8s pod uid on OS %s; please set SW_K8S_POD_UID", runtime.GOOS)
		return "", errors.New("cannot determine k8s pod uid on host OS")
	}
}

func getPodUidFromProc(fn string) (string, error) {
	f, err := os.Open(fn)
	if err != nil {
		return "", err
	}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, "kube") {
			continue
		}
		if match := uuidRegex.FindString(line); match != "" {
			return match, nil
		}
	}
	return "", fmt.Errorf("no match found in file %s", fn)
}
