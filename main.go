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
	cmd.Stdout = os.Stdout
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
func deleteSourcesJar(targetDir string) error {
	matches, err := filepath.Glob(filepath.Join(targetDir, "*-sources.*"))
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

	targetDir := filepath.Join(TMP_PATH, "target")
	err = deleteSourcesJar(targetDir)
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

func main() {

	TMP_PATH := "/Users/lama/development/go-projects/keycloak-clitool/tmp/code"
	url := flag.String("url", "", "Github url of the extension")
	flag.Parse()

	if *url == "" {
		fmt.Println("Usage: ./Keycloak-extension-cli --url=https://github.com/lamoboos223/keycloak-dummy-otp-extension")
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

	// Wait for build to complete
	if err := <-done; err != nil {
		fmt.Println("[ERROR] building extension:", err)
		return
	}

	fmt.Println("[INFO] Extension built successfully")

	// Create a channel to receive the packaging type
	packagingDone := make(chan string)
	go getPackageType(TMP_PATH, packagingDone)
	packaging := <-packagingDone
	pattern := "*." + packaging

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
	// go rebuildKeycloakInstance(done)
	// if err := <-done; err != nil {
	// 	fmt.Println("[ERROR] rebuilding Keycloak instance:", err)
	// 	return
	// }

	fmt.Println("[INFO] Keycloak instance rebuilt")

	// Restart Keycloak service
	// go restartKeycloak(done)
	// if err := <-done; err != nil {
	// 	fmt.Println("[ERROR] restarting Keycloak service:", err)
	// 	return
	// }

	fmt.Println("[INFO] Keycloak service restarted")
}
