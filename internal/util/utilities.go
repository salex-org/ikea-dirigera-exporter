package util

import (
	"fmt"
	"os"
)

var Version = "dev"

func ReadEnvVar(name string) string {
	value, present := os.LookupEnv(name)
	if !present {
		fmt.Printf("Warning: Environment variable %s not set.\n", name)
	}
	return value
}

func ReadEnvVarWithDefault(name, defaultValue string) string {
	value, present := os.LookupEnv(name)
	if !present {
		return defaultValue
	} else {
		return value
	}
}
