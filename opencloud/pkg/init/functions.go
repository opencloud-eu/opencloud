package init

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/generators"
)

func checkConfigPath(configPath string) error {
	targetPath := path.Join(configPath, configFilename)
	if _, err := os.Stat(targetPath); err == nil {
		return fmt.Errorf("config in %s already exists", targetPath)
	}
	return nil
}

func configExists(configPath string) bool {
	targetPath := path.Join(configPath, configFilename)
	if _, err := os.Stat(targetPath); err == nil {
		return true
	}
	return false
}

func backupOpenCloudConfigFile(configPath string) (string, error) {
	sourceConfig := path.Join(configPath, configFilename)
	targetBackupConfig := path.Join(configPath, configFilename+"."+time.Now().Format("2006-01-02-15-04-05")+".backup")
	source, err := os.Open(sourceConfig)
	if err != nil {
		log.Fatalf("Could not read %s (%s)", sourceConfig, err)
	}
	defer source.Close()
	target, err := os.Create(targetBackupConfig)
	if err != nil {
		log.Fatalf("Could not generate backup %s (%s)", targetBackupConfig, err)
	}
	defer target.Close()
	_, err = io.Copy(target, source)
	if err != nil {
		log.Fatalf("Could not write backup %s (%s)", targetBackupConfig, err)
	}
	return targetBackupConfig, nil
}

// printBanner prints the generated opencloud config banner.
func printBanner(targetPath, ocAdminServicePassword, targetBackupConfig string) {
	fmt.Printf(
		"\n=========================================\n"+
			" generated OpenCloud Config\n"+
			"=========================================\n"+
			" configpath : %s\n"+
			" user       : admin\n"+
			" password   : %s\n\n",
		targetPath, ocAdminServicePassword)
	if targetBackupConfig != "" {
		fmt.Printf("\n=========================================\n"+
			"An older config file has been backed up to\n %s\n\n",
			targetBackupConfig)
	}
}

// writeConfig writes the config to the target path and prints a banner
func writeConfig(configPath, ocAdminServicePassword, targetBackupConfig string, yamlOutput []byte) error {
	targetPath := path.Join(configPath, configFilename)
	err := os.WriteFile(targetPath, yamlOutput, 0600)
	if err != nil {
		return err
	}
	printBanner(targetPath, ocAdminServicePassword, targetBackupConfig)
	return nil
}

// writePatch writes the diff to a file
func writePatch(configPath string, yamlOutput []byte) error {
	fmt.Println("running in diff mode")
	tmpFile := path.Join(configPath, "opencloud.yaml.tmp")
	err := os.WriteFile(tmpFile, yamlOutput, 0600)
	if err != nil {
		return err
	}
	fmt.Println("diff -u " + path.Join(configPath, configFilename) + " " + tmpFile)
	cmd := exec.Command("diff", "-u", path.Join(configPath, configFilename), tmpFile)
	stdout, err := cmd.Output()
	if err == nil {
		err = os.Remove(tmpFile)
		if err != nil {
			return err
		}
		fmt.Println("no changes, your config is up to date")
		return nil
	}
	fmt.Println(string(stdout))
	err = os.Remove(tmpFile)
	if err != nil {
		return err
	}
	patchPath := path.Join(configPath, "opencloud.config.patch")
	err = os.WriteFile(patchPath, stdout, 0600)
	if err != nil {
		return err
	}
	fmt.Printf("diff written to %s\n", patchPath)
	return nil
}

func secretFromEnvOrRandom(envVar string, length int) (string, error) {
	if v := os.Getenv(envVar); v != "" {
		return v, nil
	}
	return generators.GenerateRandomPassword(length)
}
