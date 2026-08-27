package ssh

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/VitorAllux/devtools/internal/config"
	"github.com/VitorAllux/devtools/internal/run"
)

type Manager struct {
	Config *config.Config
	Runner run.Runner
	Store  Store

	connectionProbe func(context.Context, string) error
}

type PrepareResult struct {
	RestoredBackup  bool
	WarningMessages []string
}

type SkippedBackupError struct {
	Reason string
}

func (e SkippedBackupError) Error() string {
	return e.Reason
}

func IsSkippedBackup(err error) bool {
	var skipped SkippedBackupError
	return errors.As(err, &skipped)
}

func NewManager(cfg *config.Config, runner run.Runner) *Manager {
	return &Manager{
		Config: cfg,
		Runner: runner,
		Store:  Store{Path: cfg.ServersFile},
	}
}

func (m *Manager) Prepare(ctx context.Context) (PrepareResult, error) {
	result := PrepareResult{}
	hasEntries, err := m.Store.HasEntries()
	if err != nil {
		return result, err
	}
	if hasEntries {
		return result, nil
	}

	if fileHasContent(m.Config.EncryptedServersFile) {
		if _, err := m.Runner.LookPath("age"); err != nil {
			result.WarningMessages = append(result.WarningMessages, "encrypted SSH backup exists, but age is not installed")
			return result, nil
		}
		if !fileHasContent(m.Config.AgeKeyFile) {
			result.WarningMessages = append(result.WarningMessages, "encrypted SSH backup exists, but the AGE key file is missing")
			return result, nil
		}
		if err := m.decryptBackup(ctx); err != nil {
			result.WarningMessages = append(result.WarningMessages, fmt.Sprintf("could not restore encrypted SSH backup: %v", err))
			return result, nil
		}
		result.RestoredBackup = true
	}

	return result, nil
}

func (m *Manager) SyncBackup(ctx context.Context) error {
	if _, err := m.Runner.LookPath("age"); err != nil {
		return SkippedBackupError{Reason: "age is not installed; encrypted SSH backup was not updated"}
	}
	if !fileHasContent(m.Config.AgeRecipientsFile) {
		return SkippedBackupError{Reason: "AGE recipients file is missing or empty; encrypted SSH backup was not updated"}
	}
	hasEntries, err := m.Store.HasEntries()
	if err != nil {
		return err
	}
	if !hasEntries {
		return SkippedBackupError{Reason: "SSH list is empty; encrypted SSH backup was not updated"}
	}

	if err := os.MkdirAll(filepath.Dir(m.Config.EncryptedServersFile), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(m.Config.EncryptedServersFile), ".servers.list.age.*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Remove(tmpName); err != nil {
		return err
	}
	defer os.Remove(tmpName)

	if err := m.Runner.Run(ctx, "", "age", "-R", m.Config.AgeRecipientsFile, "-o", tmpName, m.Config.ServersFile); err != nil {
		return err
	}
	return os.Rename(tmpName, m.Config.EncryptedServersFile)
}

func (m *Manager) RestoreBackup(ctx context.Context) error {
	return m.decryptBackup(ctx)
}

func (m *Manager) decryptBackup(ctx context.Context) error {
	if err := os.MkdirAll(filepath.Dir(m.Config.ServersFile), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(m.Config.ServersFile), ".servers.list.decrypt.*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Remove(tmpName); err != nil {
		return err
	}
	defer os.Remove(tmpName)

	if err := m.Runner.Run(ctx, "", "age", "-d", "-i", m.Config.AgeKeyFile, "-o", tmpName, m.Config.EncryptedServersFile); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, 0o600); err != nil {
		return err
	}
	return os.Rename(tmpName, m.Config.ServersFile)
}

func fileHasContent(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir() && info.Size() > 0
}
