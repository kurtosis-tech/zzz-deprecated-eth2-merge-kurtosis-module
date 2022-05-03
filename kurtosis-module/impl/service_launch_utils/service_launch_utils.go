package service_launch_utils

import (
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
	"github.com/kurtosis-tech/stacktrace"
	"os"
	"text/template"
)

func FillTemplateToPath(tmpl *template.Template, data interface{}, destFilepath string) error {
	destFp, err := os.Create(destFilepath)
	if err != nil {
		return stacktrace.Propagate(err, "An error occurred creating file with filepath '%v' on the module container", destFilepath)
	}
	defer destFp.Close()
	if err := tmpl.Execute(destFp, data); err != nil {
		return stacktrace.Propagate(err, "An error occurred filling the template to destination '%v'", destFilepath)
	}
	return nil
}

func FillTemplateToSharedPath(tmpl *template.Template, data interface{}, destination *services.SharedPath) error {
	destFilepath := destination.GetAbsPathOnThisContainer()
	destFp, err := os.Create(destFilepath)
	if err != nil {
		return stacktrace.Propagate(err, "An error occurred creating file with filepath '%v' on the module container", destFilepath)
	}
	if err := tmpl.Execute(destFp, data); err != nil {
		return stacktrace.Propagate(err, "An error occurred filling the template to destination '%v'", destFilepath)
	}
	return nil
}
