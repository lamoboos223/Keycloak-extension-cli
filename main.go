package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/beevik/etree"
)

// Git clone function
func gitClone(repoURL, destination string, done chan<- error) {
	cmd := exec.Command("git", "clone", repoURL, destination)
	// cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	done <- err
}

// Get package type from pom.xml
func getPackageType(TMP_PATH string, done chan<- string) {
	doc := etree.NewDocument()
	if err := doc.ReadFromFile(TMP_PATH + "/pom.xml"); err != nil {
		fmt.Println("Error reading pom.xml:", err)
		log.Fatal()
	}

	packaging := doc.FindElement("//project/packaging")
	if packaging == nil {
		fmt.Println("Packaging element not found in pom.xml")
		log.Fatal()
	}

	done <- packaging.Text()
}

// Delete sources JAR files from the target directory
func deleteFile(targetDir string) error {
	matches, err := filepath.Glob(targetDir)
	if err != nil {
		return err
	}

	for _, match := range matches {
		err := os.Remove(match)
		if err != nil {
			return err
		}
	}

	return nil
}

// Build extension
func buildExtension(TMP_PATH string, done chan<- error) {
	cmd := exec.Command("mvn", "clean", "package", "-Dmaven.test.skip=true", "-Dclassifier=")
	cmd.Dir = TMP_PATH
	cmd.Stderr = os.Stderr
	fmt.Println("[INFO] Running Maven command: mvn clean package")

	err := cmd.Run()
	if err != nil {
		done <- fmt.Errorf("[ERROR] running Maven command: %v", err)
		return
	}
	fmt.Println("[INFO] Maven build completed successfully")

	// Delete sources JAR files from the target directorys
	targetDir := filepath.Join(TMP_PATH, "target")
	err = deleteFile(filepath.Join(targetDir, "*-sources.*"))
	if err != nil {
		done <- fmt.Errorf("[ERROR] deleting sources JAR file: %v", err)
		return
	}
	fmt.Println("[INFO] Deleted *-sources.* file from target directory")

	done <- nil
}

// Copy the built file to the Keycloak providers directory
func copyBuildToProviders(extensionFile string, done chan<- error) {
	KEYCLOAK_PATH := os.Getenv("KEYCLOAK_PATH")
	KEYCLOAK_PROVIDERS_PATH := filepath.Join(KEYCLOAK_PATH, "providers")
	sourceFile, err := os.Open(extensionFile)
	if err != nil {
		done <- err
		return
	}
	defer sourceFile.Close()

	destinationFile, err := os.Create(filepath.Join(KEYCLOAK_PROVIDERS_PATH, extensionFile))
	if err != nil {
		done <- err
		return
	}
	defer destinationFile.Close()

	_, err = io.Copy(destinationFile, sourceFile)
	if err != nil {
		done <- err
		return
	}

	err = destinationFile.Sync()
	if err != nil {
		done <- err
		return
	}

	fmt.Println("File copied successfully.")
	done <- nil
}

// Rebuild Keycloak instance
func rebuildKeycloakInstance(done chan<- error) {
	fmt.Println("[INFO] Rebuild Keycloak ...")
	KEYCLOAK_PATH := os.Getenv("KEYCLOAK_PATH")
	cmd := exec.Command("bin/kc.sh", "build")
	cmd.Dir = KEYCLOAK_PATH
	output, err := cmd.CombinedOutput()
	if err != nil {
		done <- fmt.Errorf("[ERROR] running command: %v\n%s", err, output)
		return
	}

	fmt.Println(string(output))
	done <- nil
}

// Restart Keycloak service
func restartKeycloak(done chan<- error) {
	fmt.Println("[INFO] Restarting Keycloak Service")
	KEYCLOAK_PATH := os.Getenv("KEYCLOAK_PATH")
	cmd := exec.Command("systemctl", "restart", "keycloak")
	cmd.Dir = KEYCLOAK_PATH
	output, err := cmd.CombinedOutput()
	if err != nil {
		done <- fmt.Errorf("[ERROR] running command: %v\n%s", err, output)
		return
	}

	fmt.Println(string(output))
	done <- nil
}

// list all available extensions
func listKeycloakExtensions() {
	KEYCLOAK_PATH := os.Getenv("KEYCLOAK_PATH")
	extensionsPath := filepath.Join(KEYCLOAK_PATH, "providers")

	err := filepath.Walk(extensionsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("Error accessing path %q: %v\n", path, err)
			return err
		}
		if !info.IsDir() {
			fmt.Println(filepath.Base(path))
		}
		return nil
	})

	if err != nil {
		fmt.Printf("Error walking the path %q: %v\n", extensionsPath, err)
	}
}

// Uninstall an extension
func uninstallExtension(extensionName string) {
	KEYCLOAK_PATH := os.Getenv("KEYCLOAK_PATH")
	extensionPath := filepath.Join(KEYCLOAK_PATH, "providers", extensionName)
	fmt.Printf("[INFO] Uninstalling Keycloak's Extension %s ...", extensionName)
	err := deleteFile(extensionPath)
	if err != nil {
		fmt.Printf("[ERROR] Uninstalling extension file: %v", err)
		return
	}
	fmt.Printf("[INFO] Uninstalled Keycloak's Extension %s ...", extensionName)
}
func printUsage() {
	fmt.Println("Keycloak Extension CLI - A tool for managing Keycloak extensions")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  Keycloak-extension-cli <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  install     Install a Keycloak extension")
	fmt.Println("  uninstall   Uninstall a Keycloak extension")
	fmt.Println("  list        List installed Keycloak extensions")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -h, --help  Show help for a command")
	fmt.Println()
	fmt.Println("Use 'Keycloak-extension-cli <command> --help' for more information about a command.")
}
func printInstallUsage() {
	fmt.Println("Install a Keycloak extension")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  Keycloak-extension-cli install --url=<github-url>")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --url    GitHub URL of the extension to install")
}
func printUninstallUsage() {
	fmt.Println("Uninstall a Keycloak extension")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  Keycloak-extension-cli uninstall --file=<jar-file>")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --file   JAR file of the extension to uninstall")
}
func printListUsage() {
	fmt.Println("List installed Keycloak extensions")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  Keycloak-extension-cli list")
	fmt.Println()
	fmt.Println("This command lists all installed Keycloak extensions.")
}
func main() {
	TMP_PATH := "/tmp/code"
	if len(os.Args) < 2 || os.Args[1] == "--help" || os.Args[1] == "-h" {
		printUsage()
		os.Exit(0)
	}

	command := os.Args[1]

	switch command {
	case "install":
		installCommand := flag.NewFlagSet("install", flag.ExitOnError)
		url := installCommand.String("url", "", "Github url of the extension")

		installCommand.Parse(os.Args[2:])

		if *url == "" {
			fmt.Println("Usage: ./Keycloak-extension-cli install --url=https://github.com/lamoboos223/keycloak-dummy-otp-extension")
			log.Fatal()
		}
		fmt.Printf("[INFO] Start downloading keycloak's extension [%s] ...\n", *url)

		// Create a channel for error handling
		done := make(chan error)

		// Clone the Git repository
		go gitClone(*url, TMP_PATH, done)

		// Wait for cloning to finish
		if err := <-done; err != nil {
			fmt.Println("[ERROR] cloning repository:", err)
			return
		}

		fmt.Println("[INFO] Cloning completed")

		// Build the extension
		go buildExtension(TMP_PATH, done)

		// Create a channel to receive the packaging type
		packagingDone := make(chan string)
		go getPackageType(TMP_PATH, packagingDone)
		packaging := <-packagingDone
		pattern := "*." + packaging

		// Wait for build to complete
		if err := <-done; err != nil {
			fmt.Println("[ERROR] building extension:", err)
			return
		}

		fmt.Println("[INFO] Extension built successfully")

		err := os.Chdir(filepath.Join(TMP_PATH, "target"))
		if err != nil {
			log.Fatalf("Error changing directory to %s: %v", TMP_PATH, err)
		}

		matches, err := filepath.Glob(pattern)
		if err != nil {
			log.Fatalf("Error finding build file: %v", err)
		}
		if len(matches) != 1 {
			log.Fatalf("Expected exactly one build file, found %d", len(matches))
		}

		buildFile := matches[0]

		// Copy the built file to the Keycloak providers directory
		go copyBuildToProviders(buildFile, done)
		if err := <-done; err != nil {
			fmt.Println("[ERROR] copying build file:", err)
			return
		}

		fmt.Println("[INFO] File copied successfully")

		// Rebuild Keycloak instance
		go rebuildKeycloakInstance(done)
		if err := <-done; err != nil {
			fmt.Println("[ERROR] rebuilding Keycloak instance:", err)
			return
		}

		fmt.Println("[INFO] Keycloak instance rebuilt")

		// Restart Keycloak service
		go restartKeycloak(done)
		if err := <-done; err != nil {
			fmt.Println("[ERROR] restarting Keycloak service:", err)
			return
		}

		fmt.Println("[INFO] Keycloak service restarted")
	case "uninstall":
		uninstallCommand := flag.NewFlagSet("uninstall", flag.ExitOnError)
		file := uninstallCommand.String("file", "", "JAR file to uninstall")

		uninstallCommand.Parse(os.Args[2:])

		if *file == "" {
			fmt.Println("Usage: ./Keycloak-extension-cli uninstall --file=test.jar")
			log.Fatal()
		}
		uninstallExtension(*file)
		fmt.Printf("Uninstalling extension %s\n", *file)
		// Rebuild Keycloak instance
		// Create a channel for error handling
		done := make(chan error)
		go rebuildKeycloakInstance(done)
		if err := <-done; err != nil {
			fmt.Println("[ERROR] rebuilding Keycloak instance:", err)
			return
		}

		fmt.Println("[INFO] Keycloak instance rebuilt")

		// Restart Keycloak service
		go restartKeycloak(done)
		if err := <-done; err != nil {
			fmt.Println("[ERROR] restarting Keycloak service:", err)
			return
		}

		fmt.Println("[INFO] Keycloak service restarted")
	case "list":
		if len(os.Args) > 2 && (os.Args[2] == "--help" || os.Args[2] == "-h") {
			printListUsage()
			os.Exit(0)
		}

		// Your list logic here
		fmt.Println("Listing installed extensions:")
		listKeycloakExtensions()

	default:
		fmt.Printf("Unknown command: %s\n", command)
		fmt.Println("Usage: ./Keycloak-extension-cli <command> [arguments]")
		fmt.Println("Commands: install, uninstall")
		os.Exit(1)
	}
	if len(os.Args) > 2 && (os.Args[2] == "--help" || os.Args[2] == "-h") {
		switch command {
		case "install":
			printInstallUsage()
		case "uninstall":
			printUninstallUsage()
		case "list":
			printListUsage()
		}
		os.Exit(0)
	}
}
