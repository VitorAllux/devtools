package run

import "context"

func Quiet(ctx context.Context, runner Runner, dir string, name string, args ...string) error {
	_, err := runner.Output(ctx, dir, name, args...)
	return err
}
