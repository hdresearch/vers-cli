package rootfs

import (
	"context"

	vers "github.com/hdresearch/vers-sdk-go"
	"github.com/hdresearch/vers-sdk-go/option"
)

func List(ctx context.Context, client *vers.Client) ([]string, error) {
	resp, err := client.API.Rootfs.List(ctx)
	if err != nil {
		return nil, err
	}
	return resp.Data.RootfsNames, nil
}

func Delete(ctx context.Context, client *vers.Client, name string) (string, error) {
	resp, err := client.API.Rootfs.Delete(ctx, name)
	if err != nil {
		return "", err
	}
	return resp.Data.RootfsName, nil
}

func Upload(ctx context.Context, client *vers.Client, name string, dockerfile string, tar []byte) (string, error) {
	body := vers.APIRootfUploadParams{Dockerfile: vers.F(dockerfile)}
	opt := option.WithRequestBody("application/x-tar", tar)
	resp, err := client.API.Rootfs.Upload(ctx, name, body, opt)
	if err != nil {
		return "", err
	}
	return resp.Data.RootfsName, nil
}
