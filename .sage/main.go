package main

import (
	"context"
	"path/filepath"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/sgtool"
	"go.einride.tech/sage/tools/sgbuf"
	"go.einride.tech/sage/tools/sgconvco"
	"go.einride.tech/sage/tools/sggit"
	"go.einride.tech/sage/tools/sggo"
	"go.einride.tech/sage/tools/sggolangcilint"
	"go.einride.tech/sage/tools/sggoreview"
	"go.einride.tech/sage/tools/sggosemanticrelease"
	"go.einride.tech/sage/tools/sgmdformat"
	"go.einride.tech/sage/tools/sgyamlfmt"
)

func main() {
	sg.GenerateMakefiles(
		sg.Makefile{
			Path:          sg.FromGitRoot("Makefile"),
			DefaultTarget: Default,
		},
	)
}

func Default(ctx context.Context) error {
	sg.Deps(ctx, ConvcoCheck, GoLint, GoReview, GoTest, FormatMarkdown, FormatYAML)
	sg.Deps(ctx, BufLint, BufGenerate)
	sg.SerialDeps(ctx, GoModTidy, GitVerifyNoDiff)
	return nil
}

func FormatYAML(ctx context.Context) error {
	sg.Logger(ctx).Println("formatting YAML files...")
	return sgyamlfmt.Run(ctx)
}

func GoModTidy(ctx context.Context) error {
	sg.Logger(ctx).Println("tidying Go module files...")
	return sg.Command(ctx, "go", "mod", "tidy", "-v").Run()
}

func GoTest(ctx context.Context) error {
	sg.Logger(ctx).Println("running Go tests...")
	return sggo.TestCommand(ctx).Run()
}

func GoReview(ctx context.Context) error {
	sg.Logger(ctx).Println("reviewing Go files...")
	return sggoreview.Run(ctx)
}

func GoLint(ctx context.Context) error {
	sg.Logger(ctx).Println("linting Go files...")
	return sggolangcilint.Run(ctx)
}

func FormatMarkdown(ctx context.Context) error {
	sg.Logger(ctx).Println("formatting Markdown files...")
	return sgmdformat.Command(ctx).Run()
}

func ConvcoCheck(ctx context.Context) error {
	sg.Logger(ctx).Println("checking git commits...")
	return sgconvco.Command(ctx, "check", "origin/master..HEAD").Run()
}

func GitVerifyNoDiff(ctx context.Context) error {
	sg.Logger(ctx).Println("verifying that git has no diff...")
	return sggit.VerifyNoDiff(ctx)
}

func BufGenerate(ctx context.Context) error {
	sg.Deps(ctx, ProtocGenGo)
	protoPath := sg.FromGitRoot("internal", "examples", "proto")
	genPath := filepath.Join(protoPath, "gen")
	if err := sg.Command(ctx, "git", "clean", "-fdx", genPath).Run(); err != nil {
		return err
	}
	cmd := sgbuf.Command(ctx, "generate", "--path", "einride")
	cmd.Dir = protoPath
	return cmd.Run()
}

func BufLint(ctx context.Context) error {
	cmd := sgbuf.Command(ctx, "lint")
	cmd.Dir = sg.FromGitRoot("internal", "examples", "proto")
	return cmd.Run()
}

func ProtocGenGo(ctx context.Context) error {
	_, err := sgtool.GoInstallWithModfile(ctx, "google.golang.org/protobuf/cmd/protoc-gen-go", sg.FromGitRoot("go.mod"))
	return err
}

func SemanticRelease(ctx context.Context, repo string, dry bool) error {
	sg.Logger(ctx).Println("triggering release...")
	args := []string{
		"--allow-initial-development-versions",
		"--allow-no-changes",
		"--ci-condition=default",
		"--provider=github",
		"--provider-opt=slug=" + repo,
	}
	if dry {
		args = append(args, "--dry")
	}
	return sggosemanticrelease.Command(ctx, args...).Run()
}
