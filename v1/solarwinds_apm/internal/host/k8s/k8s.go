package k8s

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/solarwindscloud/solarwinds-apm-go/v1/solarwinds_apm/internal/log"
	"os"
	"regexp"
	"runtime"
)

const (
	linuxNamespaceFile   = "/run/secrets/kubernetes.io/serviceaccount/namespace"
	linuxProcMountInfo   = "/proc/self/mountinfo"
	windowsNamespaceFile = `C:\var\run\secrets\kubernetes.io\serviceaccount\namespace`
)

var uuidRegex = regexp.MustCompile("[[:xdigit:]]{8}-([[:xdigit:]]{4}-){3}[[:xdigit:]]{12}")

func getNamespace(fallbackFile string) (string, error) {
	if ns, ok := os.LookupEnv("SW_K8S_POD_NAMESPACE"); ok {
		log.Debug("Successfully read k8s namespace from SW_K8S_POD_NAMESPACE")
		return ns, nil
	}

	log.Debugf("Attempting to read namespace from %s", fallbackFile)
	if ns, err := os.ReadFile(fallbackFile); err != nil {
		return "", err
	} else {
		return string(ns), nil
	}
}

func getPodname() (string, error) {
	if pn, ok := os.LookupEnv("SW_K8S_POD_NAME"); ok {
		log.Debug("Successfully read k8s pod name from SW_K8S_POD_NAME")
		return pn, nil
	}

	return os.Hostname()
}

func getPodUid(fallbackFile string) (string, error) {
	if uid, ok := os.LookupEnv("SW_K8S_POD_UID"); ok {
		log.Debug("Successfully read k8s pod uid from SW_K8S_POD_UID")
		return uid, nil
	}

	//goland:noinspection GoBoolExpressions
	if runtime.GOOS == "linux" {
		return getPodUidFromProc(fallbackFile)
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
		if match := uuidRegex.FindString(line); match != "" {
			return match, nil
		}
	}
	return "", fmt.Errorf("no match found in file %s", fn)
}
