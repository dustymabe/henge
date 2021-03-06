package util

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang/glog"
	"github.com/openshift/source-to-image/pkg/api"
)

// FixInjectionsWithRelativePath fixes the injections that does not specify the
// destination directory or the directory is relative to use the provided
// working directory.
func FixInjectionsWithRelativePath(workdir string, injections *api.VolumeList) {
	if len(*injections) == 0 {
		return
	}
	newList := api.VolumeList{}
	for _, injection := range *injections {
		changed := false
		if filepath.Clean(injection.Destination) == "." {
			injection.Destination = workdir
			changed = true
		}
		if !filepath.IsAbs(injection.Destination) {
			injection.Destination = filepath.Join(workdir, injection.Destination)
			changed = true
		}
		if changed {
			glog.V(5).Infof("Using %q as a destination for injecting %q", injection.Destination, injection.Source)
		}
		newList = append(newList, injection)
	}
	*injections = newList
}

// ExpandInjectedFiles returns a flat list of all files that are injected into a
// container. All files from nested directories are returned in the list.
func ExpandInjectedFiles(injections api.VolumeList) ([]string, error) {
	result := []string{}
	for _, s := range injections {
		if _, err := os.Stat(s.Source); err != nil {
			return nil, err
		}
		err := filepath.Walk(s.Source, func(path string, f os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if f.IsDir() {
				return nil
			}
			newPath := filepath.Join(s.Destination, strings.TrimPrefix(path, s.Source))
			result = append(result, newPath)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

// CreateInjectedFilesRemovalScript creates a shell script that contains truncation
// of all files we injected into the container. The path to the script is returned.
// When the scriptName is provided, it is also truncated together with all
// secrets.
func CreateInjectedFilesRemovalScript(files []string, scriptName string) (string, error) {
	rmScript := "set -e\n"
	for _, s := range files {
		rmScript += fmt.Sprintf("truncate -s0 %q\n", s)
	}

	f, err := ioutil.TempFile("", "s2i-injection-remove")
	if err != nil {
		return "", err
	}
	if len(scriptName) > 0 {
		rmScript += fmt.Sprintf("truncate -s0 %q\n", scriptName)
	}
	rmScript += "set +e\n"
	err = ioutil.WriteFile(f.Name(), []byte(rmScript), 0700)
	return f.Name(), err
}

// HandleInjectionError handles the error caused by injection and provide
// reasonable suggestion to users.
func HandleInjectionError(p api.VolumeSpec, err error) error {
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), "no such file or directory") {
		glog.Errorf("The destination directory for %q injection must exist in container (%q)", p.Source, p.Destination)
		return err
	}
	glog.Errorf("Error occured during injecting %q to %q: %v", p.Source, p.Destination, err)
	return err
}
