package shared

import (
	"errors"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// CheckAccount chan chay nham account: credentials phai trung expectedAccount cua stack.
// Khong dua gia tri account vao loi vi log CI cua repo la public.
func CheckAccount(actual, expected string) error {
	if expected == "" {
		return errors.New("config expectedAccount is empty")
	}
	if actual != expected {
		return errors.New("aws credentials belong to a different account than expectedAccount of this stack")
	}
	return nil
}

// ImportOpt tra ve option import khi tiep nhan resource dang ton tai (id != ""), rong neu tao moi.
// Sau khi up import thanh cong, tat importExisting de bo option nay theo khuyen nghi cua Pulumi.
func ImportOpt(id string) []pulumi.ResourceOption {
	if id == "" {
		return nil
	}
	return []pulumi.ResourceOption{pulumi.Import(pulumi.ID(id))}
}
