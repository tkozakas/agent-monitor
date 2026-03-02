package client

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	delegationsDir    = "delegations"
	delegationFileExt = ".md"
	defaultStatus     = "unknown"

	fieldTitle       = "title:"
	fieldDescription = "description:"
	fieldAgent       = "agent:"
	fieldStatus      = "status:"
)

// Delegation represents a delegated task read from the opencode state directory.
type Delegation struct {
	ID          string
	Agent       string
	Status      string
	Title       string
	Description string
	StartedAt   time.Time
	CompletedAt *time.Time
	FilePath    string
}

// ReadDelegations reads all delegation files from the opencode state directory.
func (c *Client) ReadDelegations() ([]Delegation, error) {
	stateDir := stateDirectory()
	if stateDir == "" {
		return nil, nil
	}

	baseDir := filepath.Join(stateDir, opencodeDir, delegationsDir)
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		return nil, nil
	}

	var delegations []Delegation
	err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, delegationFileExt) {
			return err
		}

		d, err := parseDelegationFile(path, info)
		if err != nil {
			return nil
		}
		delegations = append(delegations, d)
		return nil
	})

	return delegations, err
}

func parseDelegationFile(path string, info os.FileInfo) (Delegation, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Delegation{}, err
	}

	content := string(data)
	d := Delegation{
		ID:        strings.TrimSuffix(info.Name(), delegationFileExt),
		FilePath:  path,
		StartedAt: info.ModTime(),
	}

	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, fieldTitle):
			d.Title = strings.TrimSpace(strings.TrimPrefix(line, fieldTitle))
		case strings.HasPrefix(line, fieldDescription):
			d.Description = strings.TrimSpace(strings.TrimPrefix(line, fieldDescription))
		case strings.HasPrefix(line, fieldAgent):
			d.Agent = strings.TrimSpace(strings.TrimPrefix(line, fieldAgent))
		case strings.HasPrefix(line, fieldStatus):
			d.Status = strings.TrimSpace(strings.TrimPrefix(line, fieldStatus))
		}
	}

	if d.Status == "" {
		d.Status = defaultStatus
	}

	return d, nil
}
