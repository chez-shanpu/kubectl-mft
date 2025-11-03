package mft

import (
	"context"
	"fmt"
	"os"
)

type Repository interface {
	Dump(ctx context.Context) ([]byte, error)
	Save(ctx context.Context, manifestPath string) error
	Path(ctx context.Context) (string, error)
	Push(ctx context.Context) error
	Pull(ctx context.Context) error
}

// Dump retrieves and outputs a manifest from local OCI layout storage
func Dump(ctx context.Context, r Repository, filePath string) error {
	data, err := r.Dump(ctx)
	if err != nil {
		return err
	}

	if filePath == "" {
		fmt.Print(string(data))
		return nil
	}

	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = out.Write(data)
	if err != nil {
		return err
	}
	fmt.Println(filePath)
	return nil
}

// Pack packages a Kubernetes manifest into OCI layout format
func Pack(ctx context.Context, r Repository, manifest string) error {
	return r.Save(ctx, manifest)
}

func Path(ctx context.Context, r Repository) error {
	path, err := r.Path(ctx)
	if err != nil {
		return err
	}
	fmt.Println(path)
	return nil
}

// Pull pulls a Kubernetes manifest from an OCI registry
func Pull(ctx context.Context, r Repository) error {
	return r.Pull(ctx)
}

// Push pushes a Kubernetes manifest to an OCI registry
func Push(ctx context.Context, r Repository) error {
	return r.Push(ctx)
}
