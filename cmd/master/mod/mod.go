package mod

import (
	"bytes"
	"encoding/hex"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/vechain/solidb/cmd/master/draft"
	"github.com/vechain/solidb/crypto"
	"github.com/vechain/solidb/spec"
	"github.com/vechain/solidb/utils/fpath"
	yaml "gopkg.in/yaml.v2"
)

const (
	mainFileName  = ".solidb.master"
	draftFileName = "draft.yaml"
)

// spec stages
const (
	StageProposed = "proposed"
	StageApproved = "approved"
)

func makeStageFileName(stage string) string {
	return "." + stage + ".conf"
}

type mainFileData struct {
	Key string
}

// Model manages files of cluster master
type Model struct {
	dir      string
	identity *crypto.Identity

	draft *draft.Draft
}

// New create a new model instance
func New(dir string, replicas int) (*Model, error) {
	draft, err := draft.New(replicas)
	if err != nil {
		return nil, err
	}
	identity, err := crypto.GenerateIdentity()
	if err != nil {
		return nil, errors.Wrap(err, "new model")
	}
	return &Model{
		dir:      dir,
		identity: identity,
		draft:    draft,
	}, nil
}

// Load load a existed model from dir
func Load(dir string) (*Model, error) {

	md, err := loadMainFile(dir)
	if err != nil {
		return nil, err
	}
	key, err := hex.DecodeString(md.Key)
	if err != nil {
		return nil, err
	}
	identity, err := crypto.NewIdentity(key)
	if err != nil {
		return nil, err
	}

	draftFilePath := filepath.Join(dir, draftFileName)
	data, err := ioutil.ReadFile(draftFilePath)
	if err != nil {
		return nil, err
	}
	draft, err := draft.Unmarshal(data)
	if err != nil {
		return nil, err
	}

	m := Model{
		dir:      dir,
		identity: identity,
		draft:    draft,
	}

	return &m, nil
}

// Current load model from current working dir
func Current() (*Model, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return Load(dir)
}

func loadMainFile(dir string) (*mainFileData, error) {
	mainFilePath := filepath.Join(dir, mainFileName)
	if exists, err := fpath.PathExists(mainFilePath); err != nil {
		return nil, err
	} else if !exists {
		return nil, errors.New("not a solidb")
	}

	data, err := ioutil.ReadFile(mainFilePath)
	if err != nil {
		return nil, err
	}
	var md mainFileData
	if err := yaml.Unmarshal(data, &md); err != nil {
		return nil, err
	}
	return &md, nil
}

func (m *Model) saveMainFile() error {
	mainFilePath := filepath.Join(m.dir, mainFileName)
	if exists, err := fpath.PathExists(mainFilePath); err != nil {
		return err
	} else if !exists {
		md := mainFileData{
			Key: hex.EncodeToString(m.identity.PrivateKey()),
		}
		data, err := yaml.Marshal(&md)
		if err != nil {
			return err
		}

		if err := ioutil.WriteFile(mainFilePath, data, 0600); err != nil {
			return err
		}
	}
	return nil
}

// Draft returns draft of cluster
func (m *Model) Draft() *draft.Draft {
	return m.draft
}

// Identity returns identity of cluster master
func (m *Model) Identity() *crypto.Identity {
	return m.identity
}

// Save save model state into files
func (m *Model) Save() error {
	if err := m.saveMainFile(); err != nil {
		return err
	}

	data, err := m.draft.Marshal()
	if err != nil {
		return err
	}
	draftFilePath := filepath.Join(m.dir, draftFileName)

	return ioutil.WriteFile(draftFilePath, data, 0600)
}

// GetNode get draft node by index
func (m *Model) GetNode(index int) (*draft.Node, error) {
	if index >= 0 && index < len(m.draft.Nodes) {
		return &m.draft.Nodes[index], nil
	}
	return nil, errors.New("invalid index")
}

// AddNode add node into draft
func (m *Model) AddNode(node draft.Node) error {
	if node.Weight < 0 {
		return errors.New("node weight should be >= 0")
	}
	m.draft.Nodes = append(m.draft.Nodes, node)
	return m.draft.Validate()
}

func (m *Model) AlterNode(atIndex int, node draft.Node) error {
	if atIndex >= 0 && atIndex < len(m.draft.Nodes) {
		if node.Weight < 0 {
			return errors.New("node weight should be >= 0")
		}
		m.draft.Nodes[atIndex] = node
		return nil
	}
	return errors.New("invalid index")
}

func (m *Model) RemoveNode(index int) error {
	if index >= 0 || index < len(m.draft.Nodes) {
		m.draft.Nodes = append(m.draft.Nodes[:index], m.draft.Nodes[index+1:]...)
		return nil
	}
	return errors.New("invalid index")
}

func (m *Model) BuildSpec() (*spec.Spec, error) {
	sat, err := m.draft.Alloc()
	if err != nil {
		return nil, err
	}
	revision := 0
	proposed, err := m.LoadSpec(StageProposed)
	if err != nil {
		return nil, err
	}
	if proposed.V != nil {
		data1, _ := yaml.Marshal(sat)
		data2, _ := yaml.Marshal(&proposed.V.SAT)
		if bytes.Equal(data1, data2) {
			revision = proposed.V.Revision
		} else {
			revision = proposed.V.Revision + 1
		}
	}

	return &spec.Spec{
		Revision: revision,
		SAT:      *sat,
	}, nil
}

func (m *Model) SaveSpec(stage string, s spec.Spec) error {
	data, err := yaml.Marshal(&s)
	if err != nil {
		return err
	}
	path := filepath.Join(m.dir, makeStageFileName(stage))
	if err := ioutil.WriteFile(path, data, 0600); err != nil {
		return err
	}
	return nil
}

func (m *Model) LoadSpec(stage string) (*spec.OptSpec, error) {
	path := filepath.Join(m.dir, makeStageFileName(stage))
	if exists, err := fpath.PathExists(path); err != nil {
		return nil, err
	} else if !exists {
		return &spec.OptSpec{}, nil
	}
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	s := spec.Spec{}
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return &spec.OptSpec{V: &s}, nil
}
