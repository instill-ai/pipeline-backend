package service

import mgmtPB "github.com/instill-ai/protogen-go/vdp/mgmt/v1alpha"

func GenOwnerPermalink(owner *mgmtPB.User) string {
	return "users/" + owner.GetUid()
}
