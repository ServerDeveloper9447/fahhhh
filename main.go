package main

import (
	"embed"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/gopxl/beep"
	"github.com/gopxl/beep/mp3"
	"github.com/gopxl/beep/speaker"
	"github.com/mitchellh/go-ps"
)

//go:embed fahhhh.mp3
var dir embed.FS

const (
	startMarker = "### ERR-SOUND-START ###"
	endMarker   = "### ERR-SOUND-END ###"
)

func detectShell() string {
	pid := os.Getppid()
	p, _ := ps.FindProcess(pid)
	if p == nil {
		return "unknown"
	}
	return p.Executable()
}

func getHookForShell(shell, exec string) (string, string) {
	home, _ := os.UserHomeDir()
	dir := filepath.Dir(exec)

	switch shell {
	case "bash.exe", "git-bash.exe":
		path := filepath.Join(home, ".bashrc")
		dirSlash := filepath.ToSlash(dir)
		hook := fmt.Sprintf(`
export PATH="%s:$PATH"
alias fahhhh=fahhhh.exe
_err_sound_hook() {
  local status=$?
  if [ $status -ne 0 ]; then
    ( fahhhh play > /dev/null 2>&1 & disown)
  fi
}
PROMPT_COMMAND="_err_sound_hook; $PROMPT_COMMAND"
`, toBashDir(dirSlash))
		return path, hook

	case "bash":
		path := filepath.Join(home, ".bashrc")
		dirSlash := filepath.ToSlash(dir)
		hook := fmt.Sprintf(`
export PATH="%s:$PATH"
_err_sound_hook() {
  local status=$?
  if [ $status -ne 0 ]; then
    ( fahhhh play > /dev/null 2>&1 & disown)
  fi
}
PROMPT_COMMAND="_err_sound_hook; $PROMPT_COMMAND"
`, toBashDir(dirSlash))
		return path, hook

	case "zsh":
		path := filepath.Join(home, ".zshrc")
		dirSlash := filepath.ToSlash(dir)
		hook := fmt.Sprintf(`
export PATH="%s:$PATH"
_err_sound_hook() {
	local code=$?
	if [ $code -ne 0 ]; then
		(fahhhh play > /dev/null 2>&1 & )
	fi
}
precmd_functions+=(_err_sound_hook)
`, toBashDir(dirSlash))
		return path, hook

	case "powershell.exe":
		// PowerShell Desktop and Core use different profile paths
		path := filepath.Join(home, "Documents", "WindowsPowerShell", "Microsoft.PowerShell_profile.ps1")
		// Check if the directory exists, if not, it might be PowerShell Core
		if _, err := os.Stat(filepath.Dir(path)); os.IsNotExist(err) {
			path = filepath.Join(home, "Documents", "PowerShell", "Microsoft.PowerShell_profile.ps1")
		}

		hook := fmt.Sprintf(`
$env:PATH = "%s;" + $env:PATH
function Prompt {
	$lastCommandSucceeded = $?
    if (-not $lastCommandSucceeded) {
        Start-Process "fahhhh" -ArgumentList "play" -WindowStyle Hidden
    }
    return "PS $($ExecutionContext.SessionState.Path.CurrentLocation)> "
}
`, dir)
		return path, hook

	default:
		return "", ""
	}
}

func toBashDir(dir string) string {
	path := strings.ReplaceAll(dir, "\\", "/")
	re := regexp.MustCompile(`^([a-zA-Z]):/`)

	if re.MatchString(path) {
		path = re.ReplaceAllStringFunc(path, func(match string) string {
			drive := strings.ToLower(string(match[0]))
			return "/" + drive + "/"
		})
	}
	return path
}

func install() {
	appData := ""
	switch runtime.GOOS {
	case "windows":
		appData = os.Getenv("APPDATA")
	case "linux":
		appData = os.Getenv("XDG_DATA_HOME")
		if appData == "" {
			home, _ := os.UserHomeDir()
			appData = filepath.Join(home, ".local", "share")
		}
	}
	if appData == "" {
		fmt.Println("Cannot retrieve appdata directory.")
		return
	}
	targetDir := filepath.Join(appData, "fahhhh")
	targetPath := ""
	switch runtime.GOOS {
	case "windows":
	targetPath = filepath.Join(targetDir, "fahhhh.exe")
	case "linux":
	targetPath = filepath.Join(targetDir, "fahhhh")
	}
	
	_ = os.Mkdir(targetDir, 0755)

	selfPath, err := os.Executable()
	if err != nil {
		return
	}

	input, _ := os.ReadFile(selfPath)
	_ = os.WriteFile(targetPath, input, 0755)

	shell := detectShell()
	configPath, hook := getHookForShell(shell, targetPath)
	if configPath == "" && hook == "" {
		fmt.Println("We don't support anything other than bash, pwsh and zsh at this moment.")
		return
	}
	content, _ := os.ReadFile(configPath)
	if strings.Contains(string(content), startMarker) {
		return // Already installed
	}

	f, _ := os.OpenFile(configPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	defer f.Close()

	payload := fmt.Sprintf("\n%s\n%s\n%s\n", startMarker, hook, endMarker)
	f.WriteString(payload)
	fmt.Printf("Installed for %s. Restart your shell.", shell)
}

func uninstall() {
	appData := ""
	switch runtime.GOOS {
	case "windows":
		appData = os.Getenv("APPDATA")
	case "linux":
		appData := os.Getenv("XDG_DATA_HOME")
		if appData == "" {
			home, _ := os.UserHomeDir()
			appData = filepath.Join(home, ".local", "share")
		}
	}
	if appData == "" {
		fmt.Println("Cannot retrieve appdata directory.")
		return
	}
	targetDir := filepath.Join(appData, "fahhhh")
	targetPath := ""
	switch runtime.GOOS {
	case "windows":
	targetPath = filepath.Join(targetDir, "fahhhh.exe")
	case "linux":
	targetPath = filepath.Join(targetDir, "fahhhh")
	}
	shell := detectShell()
	configPath, _ := getHookForShell(shell, targetPath)

	content, err := os.ReadFile(configPath)
	if err != nil {
		return
	}
	lines := strings.Split(string(content), "\n")

	var newLines []string
	skipping := false

	for _, line := range lines {
		if strings.Contains(line, startMarker) {
			skipping = true
			continue
		}
		if strings.Contains(line, endMarker) {
			skipping = false
			continue
		}
		if !skipping {
			newLines = append(newLines, line)
		}
	}

	os.WriteFile(configPath, []byte(strings.TrimSpace(strings.Join(newLines, "\n"))), 0644)
	os.RemoveAll(targetDir)
	fmt.Printf("Uninstalled for %s. Restart your shell.", shell)
}

func playSound() {
	file, err := dir.Open("fahhhh.mp3")
	if err != nil {
		fmt.Println("No file found")
		return
	}
	streamer, format, err := mp3.Decode(file)
	if err != nil {
		fmt.Println("Cannot decode file")
		return
	}
	defer streamer.Close()
	speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
	done := make(chan bool)
	speaker.Play(beep.Seq(streamer, beep.Callback(func() {
		done <- true
	})))

	<-done
}

func main() {
	installCmd := flag.NewFlagSet("install", flag.ExitOnError)
	uninstallCmd := flag.NewFlagSet("uninstall", flag.ExitOnError)
	playCmd := flag.NewFlagSet("play", flag.ExitOnError)

	flag.Usage = func() {
		fmt.Printf("Usage: fahhhh <command>\n\n")
		fmt.Println("The commands are:")
		fmt.Println("  install    installs the utility for the current shell")
		fmt.Println("  uninstall  uninstalls the utility for the current shell")
		fmt.Println("  play       internal command to play the sound")
	}

	installCmd.Usage = func() {}
	uninstallCmd.Usage = func() {}
	playCmd.Usage = func() {}

	if len(os.Args) < 2 {
		flag.Usage()
		return
	}

	switch os.Args[1] {
	case "install":
		installCmd.Parse(os.Args[2:])
		install()
	case "uninstall":
		uninstallCmd.Parse(os.Args[2:])
		uninstall()
	case "play":
		playCmd.Parse(os.Args[2:])
		playSound()
	default:
		fmt.Printf("unknown command: %s\n", os.Args[1])
		flag.Usage()
		os.Exit(1)
	}
}
