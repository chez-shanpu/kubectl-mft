package mft

import (
	"context"
	"fmt"
	"os"
)

type Repository interface {
	Dump(ctx context.Context) ([]byte, error)
	Save(ctx context.Context, manifestPath string) error
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

// Pull pulls a Kubernetes manifest from an OCI registry
func Pull(ctx context.Context, r Repository) error {
	return r.Pull(ctx)
}

// Push pushes a Kubernetes manifest to an OCI registry
func Push(ctx context.Context, r Repository) error {
	return r.Push(ctx)
}
