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

func gitClone(repoURL, destination string) error {
	cmd := exec.Command("git", "clone", repoURL, destination)

	// Set output to os.Stdout or os.Stderr if you want to see git clone command output
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return err
	}

	return nil
}
func getPackageType(TMP_PATH string) string {

	doc := etree.NewDocument()
	if err := doc.ReadFromFile(TMP_PATH + "/pom.xml"); err != nil {
		fmt.Println("Error reading pom.xml:", err)
		log.Fatal()
	}

	// Find the packaging element in the XML
	packaging := doc.FindElement("//project/packaging")
	if packaging == nil {
		fmt.Println("Packaging element not found in pom.xml")
		log.Fatal()
	}

	// Get the text value of the packaging element
	return packaging.Text()
}
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
func buildExtension(TMP_PATH string) {
	// Run Maven build
	cmd := exec.Command("mvn", "clean", "package", "-Dmaven.test.skip=true", "-Dclassifier=")
	cmd.Dir = TMP_PATH

	// Redirect stdout and stderr
	// cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	fmt.Println("[INFO] Running Maven command: mvn clean package")
	err := cmd.Run()
	if err != nil {
		log.Fatalf("[ERROR] running Maven command: %v", err)
	}
	fmt.Println("[INFO] Maven build completed successfully")

	// Delete *-sources.* file from target directory
	targetDir := filepath.Join(TMP_PATH, "target")
	err = deleteSourcesJar(targetDir)
	if err != nil {
		log.Fatalf("[ERROR] deleting sources JAR file: %v", err)
	}
	fmt.Println("[INFO] Deleted *-sources.* file from target directory")
}
func copyBuildToProviders(extensionFile string) {
	KEYCLOAK_PATH := os.Getenv("KEYCLOAK_PATH")
	KEYCLOAK_PROVIDERS_PATH := filepath.Join(KEYCLOAK_PATH, "providers")
	sourceFile, err := os.Open(extensionFile)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer sourceFile.Close()

	// Create destination file
	destinationFile, err := os.Create(filepath.Join(KEYCLOAK_PROVIDERS_PATH, extensionFile))
	if err != nil {
		log.Fatal(err.Error())
	}
	defer destinationFile.Close()

	// Copy the contents from source to destination
	_, err = io.Copy(destinationFile, sourceFile)
	if err != nil {
		log.Fatal(err.Error())
	}

	// Flushes memory to disk ensuring file copy
	err = destinationFile.Sync()
	if err != nil {
		log.Fatal(err.Error())
	}

	fmt.Println("File copied successfully.")
}
func rebuildKeycloakInstance() {
	fmt.Println("[INFO] Rebuild Keycloak ...")
	KEYCLOAK_PATH := os.Getenv("KEYCLOAK_PATH")
	cmd := exec.Command("bin/kc.sh", "build")
	cmd.Dir = KEYCLOAK_PATH
	// Run the command and capture the output
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("[ERROR] running command: %v\n%s", err, output)
	}

	// Print the output if needed
	fmt.Println(string(output))
}
func restartKeycloak() {
	fmt.Println("[INFO] Restarting Keycloak Service")
	KEYCLOAK_PATH := os.Getenv("KEYCLOAK_PATH")
	cmd := exec.Command("systemctl", "restart", "keycloak")
	cmd.Dir = KEYCLOAK_PATH
	// Run the command and capture the output
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("[ERROR] running command: %v\n%s", err, output)
	}

	// Print the output if needed
	fmt.Println(string(output))
}
func main() {

	TMP_PATH := "/tmp/code"
	// Define flags
	url := flag.String("url", "", "Github url of the extension")
	flag.Parse()

	if *url == "" {
		fmt.Println("Usage: ./Keycloak-extension-cli --url=https://github.com/lamoboos223/keycloak-dummy-otp-extension")
		log.Fatal()
	}

	fmt.Printf("[INFO] Start downloading keycloak's extension [%s] ...\n", *url)
	os.Mkdir(TMP_PATH, 0755)
	// run download command in a goroutine
	err := gitClone(*url, TMP_PATH)
	if err != nil {
		fmt.Println("[ERROR] cloning repository:", err)
		return
	}
	defer fmt.Printf("[INFO] Installed extension [%s] into keycloak's custom providers.\n", *url)

	buildExtension(TMP_PATH)
	// parse pom.xml to distinguish the packging type (war/jar)
	packaging := getPackageType(TMP_PATH)
	pattern := "*." + packaging
	err = os.Chdir(filepath.Join(TMP_PATH, "target"))
	if err != nil {
		log.Fatalf("Error changing directory to %s: %v", TMP_PATH, err)
	}
	matches, err := filepath.Glob(pattern)
	if err != nil {
		log.Fatalf("Error finding JAR file: %v", err)
	}
	// Ensure exactly one JAR file is found
	if len(matches) != 1 {
		log.Fatalf("Expected exactly one JAR file, found %d", len(matches))
	}
	// Extract the file name from the path
	buildFile := matches[0]
	copyBuildToProviders(buildFile)
	rebuildKeycloakInstance()
	restartKeycloak()
}
