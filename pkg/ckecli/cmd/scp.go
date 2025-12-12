package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

func (c *SshConfig) Scp(ctx context.Context, args []string) error {
	user, remoteHost, localFilePath, remoteFilePath, direction, err := ParseSCPArgs(args)
	if err != nil {
		return err
	}

	privateKey, err := loadPrivateKey(remoteHost)
	if err != nil {
		log.Error("failed to load private key", map[string]interface{}{
			log.FnError: err,
		})
		return err
	}

	signer, err := createSignerFromKeyString(privateKey)
	if err != nil {
		return err
	}

	// Load private key authentication
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// Create a TCP connection with a timeout
	dialer := net.Dialer{Timeout: c.ConnectionTimeout}
	rawConn, err := dialer.DialContext(ctx, "tcp", remoteHost)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer rawConn.Close()

	// Perform SSH handshake
	scpConn, chans, reqs, err := ssh.NewClientConn(rawConn, remoteHost, config)
	if err != nil {
		return fmt.Errorf("failed to establish SSH connection: %w", err)
	}

	fmt.Println("SSH connection established")
	client := ssh.NewClient(scpConn, chans, reqs)
	defer client.Close()

	// Forcefully close the connection when canceling
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			// If the context is canceled, force the connection to close, and clean up this goroutine and exit.
			scpConn.Close()
			return
		case <-done:
			// If the SCP operation completes successfully, clean up this goroutine and exit.
			return
		}
	}()

	// Make SCP session
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	// Send or Receive Branch
	switch direction {
	case "Send":
		return c.scpSend(ctx, session, scpConn, localFilePath, remoteFilePath)
	case "Receive":
		return c.scpReceive(ctx, session, scpConn, localFilePath, remoteFilePath)
	default:
		return fmt.Errorf("unknown direction: %s", direction)
	}
}

func (c *SshConfig) scpReceive(ctx context.Context, session *ssh.Session, scpConn ssh.Conn, localFilePath string, remoteFilePath string) error {
	stdout, err := session.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout: %w", err)
	}

	stdin, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdin: %w", err)
	}

	// Start SCP in “from” mode (-f)
	cmd := fmt.Sprintf("scp -f %s", remoteFilePath)
	if err := session.Start(cmd); err != nil {
		return fmt.Errorf("failed to start scp -f: %w", err)
	}

	// ctx cancel → session forced termination
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			// If the context is canceled, force the connection to close, and clean up this goroutine and exit
			scpConn.Close()
			return
		case <-done:
			// If the SCP operation completes successfully, clean up this goroutine and exit.
			return
		}
	}()

	// Send the initial ACK (0 bytes)
	if _, err := stdin.Write([]byte{0}); err != nil {
		return fmt.Errorf("failed to send initial ACK: %w", err)
	}

	reader := bufio.NewReader(stdout)

	// Read `C0644 <size> <filename>\n`
	header, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read scp header: %w", err)
	}

	// Parse
	var mode string
	var size int64
	var filename string
	_, err = fmt.Sscanf(header, "C%s %d %s", &mode, &size, &filename)
	if err != nil {
		return fmt.Errorf("failed to parse scp header (%s): %w", header, err)
	}

	// Send ACK
	if _, err := stdin.Write([]byte{0}); err != nil {
		return fmt.Errorf("failed to send ACK: %w", err)
	}

	// Create local file
	localFile, err := os.Create(localFilePath)
	if err != nil {
		return fmt.Errorf("failed to create local file: %w", err)
	}
	defer localFile.Close()

	// Receive file contents
	n, err := io.CopyN(localFile, reader, size)
	if err != nil {
		return fmt.Errorf("failed to receive file: %w", err)
	}
	if n != size {
		return fmt.Errorf("file size mismatch: expected %d, got %d", size, n)
	}

	// Read the final ACK
	if _, err := reader.ReadByte(); err != nil {
		return fmt.Errorf("failed to read final ACK: %w", err)
	}

	// Send ACK to the sender
	stdin.Write([]byte{0})
	stdin.Close()
	fmt.Println("Receive completed:", localFilePath)

	// Wait session
	if err := session.Wait(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return err
	}
	return nil
}

func (c *SshConfig) scpSend(ctx context.Context, session *ssh.Session, scpConn ssh.Conn, localFilePath string, remoteFilePath string) error {
	stdin, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdin pipe: %w", err)
	}
	defer stdin.Close()

	cmd := fmt.Sprintf("scp -t %s", remoteFilePath)
	if err := session.Start(cmd); err != nil {
		return fmt.Errorf("failed to start scp -t: %w", err)
	}

	// ctx cancel → forced termination
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			// If the context is canceled, force the connection to close, and clean up this goroutine and exit
			scpConn.Close()
			return
		case <-done:
			// If the SCP operation completes successfully, clean up this goroutine and exit.
			return
		}
	}()

	file, err := os.Open(localFilePath)
	if err != nil {
		return fmt.Errorf("failed to open local file: %w", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	fmt.Fprintf(stdin, "C%04o %d %s\n",
		stat.Mode().Perm(),
		stat.Size(),
		stat.Name(),
	)

	fmt.Println("Send initiated, waiting for completion...")
	if _, err = io.Copy(stdin, file); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("failed to copy file contents: %w", err)
	}

	// Send null byte to indicate end of transfer
	fmt.Fprint(stdin, "\x00")
	stdin.Close()

	// Wait for completion
	if err := session.Wait(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("session completion error: %w", err)
	}

	fmt.Println("Send completed.")

	return nil
}

var scpParams struct {
	recursive bool
}

// scpCmd represents the scp command
var scpCmd = &cobra.Command{
	Use:   "scp [[user@]NODE1:]FILE1 ... [[user@]NODE2:]FILE2",
	Short: "copy files between hosts via scp",
	Long: `Copy files between hosts via scp.

NODE is IP address or hostname of the node.
`,

	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		well.Go(func(ctx context.Context) error {
			s := &SshConfig{
				ConnectionTimeout:     20 * time.Second,
				SessionTimeout:        30 * time.Second, // Intentionally short timeout
				KeepAliveInterval:     30 * time.Second,
				KeepAliveReplyTimeout: 5 * time.Second, // Time out of keep-alive reply from client to server
			}
			return s.Scp(ctx, args)
		})
		well.Stop()
		return well.Wait()
	},
}

func init() {
	scpCmd.Flags().BoolVarP(&scpParams.recursive, "", "r", false, "recursively copy entire directories")
	rootCmd.AddCommand(scpCmd)
}
