package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

func detectSSHNode(arg string) string {
	nodeName := arg
	if strings.Contains(arg, "@") {
		nodeName = arg[strings.Index(arg, "@")+1:]
	}
	return nodeName
}

/*
func writeToFifo(fifo string, data string) {
	f, err := os.OpenFile(fifo, os.O_WRONLY, 0600)
	if err != nil {
		log.Error("failed to open fifo", map[string]interface{}{
			log.FnError: err,
			"fifo":      fifo,
		})
		return
	}
	defer f.Close()

	_, err = f.WriteString(data)
	if err != nil {
		log.Error("failed to write to fifo", map[string]interface{}{
			log.FnError: err,
			"fifo":      fifo,
		})
	}
}
*/

/*
func sshPrivateKey(nodeName string) (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	err = os.MkdirAll(filepath.Join(usr.HomeDir, ".ssh"), 0700)
	if err != nil {
		return "", err
	}
	fifo := filepath.Join(usr.HomeDir, ".ssh", "ckecli-ssh-key-"+strconv.Itoa(os.Getpid()))
	err = syscall.Mkfifo(fifo, 0600)
	if err != nil {
		return "", err
	}

	vc, err := inf.Vault()
	if err != nil {
		return "", err
	}
	secret, err := vc.Logical().Read(cke.SSHSecret)
	if err != nil {
		return "", err
	}
	if secret == nil {
		return "", errors.New("no ssh private keys")
	}
	privKeys := secret.Data

	mykey, ok := privKeys[nodeName]
	if !ok {
		mykey = privKeys[""]
	}
	if mykey == nil {
		return "", errors.New("no ssh private key for " + nodeName)
	}

	go func() {
		// OpenSSH reads the private key file three times, it need to write key three times.
		writeToFifo(fifo, mykey.(string))
		time.Sleep(100 * time.Millisecond)
		writeToFifo(fifo, mykey.(string))
		time.Sleep(100 * time.Millisecond)
		writeToFifo(fifo, mykey.(string))
	}()

	return fifo, nil
}
*/

/*
func ssh(ctx context.Context, args []string) error {
	node := detectSSHNode(args[0])
	fifo, err := sshPrivateKey(node)
	if err != nil {
		return err
	}
	defer os.Remove(fifo)

	sshArgs := []string{
		"-i", fifo,
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "StrictHostKeyChecking=no",
		"-o", "ConnectTimeout=60",
	}
	sshArgs = append(sshArgs, args...)
	c := exec.CommandContext(ctx, "ssh", sshArgs...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}
*/

// =======================================================================

func createFifo2() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}

	fifoFilePath := filepath.Join(usr.HomeDir, ".ssh", "ckecli-ssh-key-"+strconv.Itoa(os.Getpid()))
	_, err = os.Stat(fifoFilePath)
	if os.IsExist(err) {
		return fifoFilePath, nil
	}
	if !os.IsNotExist(err) {
		return "", err
	}

	err = os.MkdirAll(filepath.Join(usr.HomeDir, ".ssh"), 0700)
	if err != nil {
		return "", err
	}

	err = syscall.Mkfifo(fifoFilePath, 0600)
	if err != nil {
		return "", err
	}

	return fifoFilePath, err
}

func getPrivateKey(nodeName string) ([]byte, error) {
	vc, err := inf.Vault()
	if err != nil {
		return nil, err
	}
	secret, err := vc.Logical().Read(cke.SSHSecret)
	if err != nil {
		return nil, err
	}
	if secret == nil {
		return nil, errors.New("no ssh private keys")
	}
	privKeys := secret.Data

	mykey, ok := privKeys[nodeName]
	if !ok {
		mykey = privKeys[""]
	}
	if mykey == nil {
		return nil, errors.New("no ssh private key for " + nodeName)
	}
	/*
		go func() {
			// OpenSSH reads the private key file three times, it need to write key three times.
			writeToFifo(fifo, mykey.(string))
			time.Sleep(100 * time.Millisecond)
			writeToFifo(fifo, mykey.(string))
			time.Sleep(100 * time.Millisecond)
			writeToFifo(fifo, mykey.(string))
		}()
	*/
	return mykey.([]byte), nil
}

func sshAgent(ctx context.Context, privateKeyFile string) (map[string]string, error) {
	myEnv := make(map[string]string)

	sshArgs := []string{
		"-s",
	}
	cmd := exec.CommandContext(ctx, "ssh-agent", sshArgs...)
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}
	line := strings.Split(string(stdoutStderr), "\n")

	partOfLine := strings.Split(line[0], ";")
	kvPair := strings.Split(partOfLine[0], "=")
	myEnv[kvPair[0]] = kvPair[1]
	err = os.Setenv(kvPair[0], kvPair[1])
	if err != nil {
		return nil, err
	}

	partOfLine = strings.Split(line[1], ";")
	kvPair = strings.Split(partOfLine[0], "=")
	myEnv[kvPair[0]] = kvPair[1]
	err = os.Setenv(kvPair[0], kvPair[1])
	if err != nil {
		return nil, err
	}

	sshArgs2 := []string{
		privateKeyFile,
	}
	cmd0 := exec.CommandContext(ctx, "ssh-add", sshArgs2...)
	_, err = cmd0.CombinedOutput()
	if err != nil {
		return nil, err
	}

	return myEnv, nil
}

func killSshAgent(ctx context.Context) error {
	sshArgs := []string{
		"-k",
	}
	cmd := exec.CommandContext(ctx, "ssh-agent", sshArgs...)
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("Error ssh-agent -k ", err)
		return err
	}
	fmt.Println("kill ssh agent :: output=", string(stdoutStderr))
	return nil
}

func writeToFifo(fifo string, data []byte) error {
	f, err := os.OpenFile(fifo, os.O_WRONLY|os.O_CREATE|os.O_APPEND, os.ModeNamedPipe)
	if err != nil {
		log.Error("failed to open fifo", map[string]interface{}{
			log.FnError: err,
			"fifo":      fifo,
		})
		return err
	}
	defer f.Close()
	if _, err = f.Write(data); err != nil {
		log.Error("failed to write to fifo", map[string]interface{}{
			log.FnError: err,
			"fifo":      fifo,
		})
		return err
	}
	return nil
}

func sshSubMain(ctx context.Context, args []string) (error, string) {
	pipeFilename, err := createFifo2()
	if err != nil {
		return err, fmt.Sprintln("createFifo2 err=", err)
	}

	node := detectSSHNode(args[0])
	pirvateKey, err := getPrivateKey(node)
	if err != nil {
		return err, fmt.Sprintln("getPrivateKey err=", err, "node=", node)
	}

	go func() {
		if _, err := sshAgent(ctx, pipeFilename); err != nil {
			fmt.Println("getPrivateKey err=", err, "node=", node)
			// ログ出力
			return
		}
	}()

	if err = writeToFifo(pipeFilename, pirvateKey); err != nil {
		return err, fmt.Sprintln("writeToFifo err=", err)
	}
	defer os.Remove(pipeFilename)
	defer killSshAgent(ctx)

	return ssh(ctx, args)
}

func ssh(ctx context.Context, args []string) (error, string) {
	sshArgs := []string{
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "StrictHostKeyChecking=no",
		"-o", "ConnectTimeout=60",
	}
	sshArgs = append(sshArgs, args...)
	c := exec.CommandContext(ctx, "ssh", sshArgs...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run(), "OK"
}

// sshCmd represents the ssh command
var sshCmd = &cobra.Command{
	Use:   "ssh [user@]NODE [COMMAND...]",
	Short: "connect to the node via ssh",
	Long: `Connect to the node via ssh.

NODE is IP address or hostname of the node to be connected.

If COMMAND is specified, it will be executed on the node.
`,

	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		well.Go(func(ctx context.Context) error {
			err, msg := sshSubMain(ctx, args)
			fmt.Println("error message =", msg)
			return err
		})
		well.Stop()
		return well.Wait()
	},
}

func init() {
	rootCmd.AddCommand(sshCmd)
}
