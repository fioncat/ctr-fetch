package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/signature"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"
)

const (
	DockerV2SchemaVersion = 2

	DockerV2Schema2MediaType      = "application/vnd.docker.distribution.manifest.v2+json"
	DockerV2Schema2LayerMediaType = "application/vnd.docker.image.rootfs.diff.tar.gzip"

	SHA256Prefix = "sha256:"
)

type PullOptions struct {
	Username string
	Password string

	Token string

	Insecure bool

	InsecurePolicy bool

	BaseDir string

	Force bool

	Stdout io.Writer
}

type PullResult struct {
	Path     string
	Manifest Manifest
}

type Manifest struct {
	SchemaVersion int     `json:"schemaVersion"`
	MediaType     string  `json:"mediaType"`
	Layers        []Layer `json:"layers"`
}

type Layer struct {
	MediaType string `json:"mediaType"`
	Size      int64  `json:"size"`
	Digest    string `json:"digest"`
}

func parseManifest(data []byte) (*Manifest, error) {
	var m Manifest
	err := json.Unmarshal(data, &m)
	if err != nil {
		return nil, fmt.Errorf("Invalid manifest format %q: %v", string(data), err)
	}

	if m.SchemaVersion != DockerV2SchemaVersion {
		return nil, fmt.Errorf("Unsupport manifest schema version: %d, expect %d", m.SchemaVersion, DockerV2SchemaVersion)
	}

	if m.MediaType != DockerV2Schema2MediaType {
		return nil, fmt.Errorf("Unsupport manifest schema: %q, expect %q", m.SchemaVersion, DockerV2Schema2MediaType)
	}

	for _, layer := range m.Layers {
		if layer.MediaType != DockerV2Schema2LayerMediaType {
			return nil, fmt.Errorf("Unsupport layer schema: %q, expect %q", layer.MediaType, DockerV2Schema2LayerMediaType)
		}
		if !strings.HasPrefix(layer.Digest, SHA256Prefix) {
			return nil, fmt.Errorf("Unsupport layer digest %q, expect sha256", layer.Digest)
		}
	}

	return &m, nil
}

func PullImage(name string, opts PullOptions) (*PullResult, error) {
	destDir, exists, err := getDestDir(name, opts.BaseDir)
	if err != nil {
		return nil, fmt.Errorf("Get base dir error: %w", err)
	}

	if exists && !opts.Force {
		manifestPath := filepath.Join(destDir, "manifest.json")
		_, err = os.Stat(manifestPath)
		if err == nil {
			var data []byte
			data, err = os.ReadFile(manifestPath)
			if err != nil {
				return nil, fmt.Errorf("Read manifest file %s: %v", manifestPath, err)
			}

			var manifest *Manifest
			manifest, err = parseManifest(data)
			if err != nil {
				return nil, err
			}

			return &PullResult{
				Path:     destDir,
				Manifest: *manifest,
			}, nil
		}
	}

	destName := fmt.Sprintf("dir:%s", destDir)
	destRef, err := alltransports.ParseImageName(destName)
	if err != nil {
		return nil, fmt.Errorf("Invalid dir %s: %v", destDir, err)
	}
	destSystemContext := &types.SystemContext{}

	srcName := fmt.Sprintf("docker://%s", name)
	srcRef, err := alltransports.ParseImageName(srcName)
	if err != nil {
		return nil, fmt.Errorf("Invalid image name %s: %w", name, err)
	}
	srcSystemContext := &types.SystemContext{}

	if opts.Username != "" && opts.Password != "" {
		srcSystemContext.DockerAuthConfig = &types.DockerAuthConfig{
			Username: opts.Username,
			Password: opts.Password,
		}
	}

	if opts.Token != "" {
		srcSystemContext.DockerBearerRegistryToken = opts.Token
	}

	if opts.Insecure {
		srcSystemContext.DockerInsecureSkipTLSVerify = types.NewOptionalBool(true)
	}

	var policy *signature.Policy
	if opts.InsecurePolicy {
		policy = &signature.Policy{Default: []signature.PolicyRequirement{signature.NewPRInsecureAcceptAnything()}}
	} else {
		policy, err = signature.DefaultPolicy(nil)
	}
	if err != nil {
		return nil, fmt.Errorf("Create pull policy error: %w", err)
	}
	policyContext, err := signature.NewPolicyContext(policy)
	if err != nil {
		return nil, fmt.Errorf("Load trusted policy error: %w", err)
	}

	ctx := context.Background()
	manifestData, err := copy.Image(ctx, policyContext, destRef, srcRef, &copy.Options{
		ReportWriter:          opts.Stdout,
		SourceCtx:             srcSystemContext,
		DestinationCtx:        destSystemContext,
		ForceManifestMIMEType: DockerV2Schema2MediaType,
	})

	manifest, err := parseManifest(manifestData)
	if err != nil {
		return nil, err
	}

	return &PullResult{
		Path:     destDir,
		Manifest: *manifest,
	}, nil
}

func getDestDir(name, baseDir string) (string, bool, error) {
	if baseDir == "" {
		tmpDir := os.TempDir()
		baseDir = filepath.Join(tmpDir, "ctr-fetch")
	}

	nameHex := hex.EncodeToString([]byte(name))

	dir := filepath.Join(baseDir, nameHex)
	stat, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(dir, os.ModePerm)
			if err != nil {
				return "", false, fmt.Errorf("Mkdir for dest error: %w", err)
			}
			return dir, false, nil
		}
		return "", false, fmt.Errorf("Stat for dest error: %w", err)
	}

	if !stat.IsDir() {
		return "", false, fmt.Errorf("Dest dir %s is not a directory", dir)
	}

	return dir, true, nil
}
