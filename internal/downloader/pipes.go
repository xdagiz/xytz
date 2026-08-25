package downloader

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
)

type procPipes struct {
	stdoutR *os.File
	stdoutW *os.File
	stderrR *os.File
	stderrW *os.File
}

func newProcPipes() (*procPipes, error) {
	p := &procPipes{}
	var err error
	p.stdoutR, p.stdoutW, err = os.Pipe()
	if err != nil {
		return nil, fmt.Errorf("pipe error: %w", err)
	}
	p.stderrR, p.stderrW, err = os.Pipe()
	if err != nil {
		_ = p.stdoutR.Close()
		_ = p.stdoutW.Close()
		return nil, fmt.Errorf("stderr pipe error: %w", err)
	}
	return p, nil
}

func (p *procPipes) wire(cmd *exec.Cmd) {
	cmd.Stdout = p.stdoutW
	cmd.Stderr = p.stderrW
}

func (p *procPipes) closeAll() {
	_ = p.stdoutR.Close()
	_ = p.stdoutW.Close()
	_ = p.stderrR.Close()
	_ = p.stderrW.Close()
}

func (p *procPipes) waitDrained(cmd *exec.Cmd, readers *sync.WaitGroup) error {
	err := cmd.Wait()
	_ = p.stdoutW.Close()
	_ = p.stderrW.Close()
	readers.Wait()
	_ = p.stdoutR.Close()
	_ = p.stderrR.Close()
	return err
}
