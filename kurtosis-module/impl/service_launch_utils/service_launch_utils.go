package service_launch_utils

import (
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
	"github.com/kurtosis-tech/stacktrace"
	"io"
	"os"
	"text/template"
)

func FillTemplateToSharedPath(tmpl *template.Template, data interface{}, destination *services.SharedPath) error {
	destFilepath := destination.GetAbsPathOnThisContainer()
	destFp, err := os.Create(destFilepath)
	if err != nil {
		return stacktrace.Propagate(err, "An error occurred opening filepath '%v' on the module container for writing the Geth genesis config YAML", destFilepath)
	}
	if err := tmpl.Execute(destFp, data); err != nil {
		return stacktrace.Propagate(err, "An error occurred filling the template to destination '%v'", destFilepath)
	}
	return nil
}

func CopyFileToSharedPath(srcFilepathOnModuleContainer string, destSharedPath *services.SharedPath) error {
	srcFp, err := os.Open(srcFilepathOnModuleContainer)
	if err != nil {
		return stacktrace.Propagate(err, "An error occurred opening source file '%v'", srcFilepathOnModuleContainer)
	}

	destFilepathOnModuleContainer := destSharedPath.GetAbsPathOnThisContainer()
	destFp, err := os.Create(destFilepathOnModuleContainer)
	if err != nil {
		return stacktrace.Propagate(err, "An error occurred creating destination file '%v'", destFilepathOnModuleContainer)
	}

	if _, err := io.Copy(destFp, srcFp); err != nil {
		return stacktrace.Propagate(
			err,
			"An error occurred copying bytes from source file '%v' to destination file '%v'",
			srcFilepathOnModuleContainer,
			destFilepathOnModuleContainer,
		)
	}
	return nil
}
