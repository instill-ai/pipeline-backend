package googlecloudstorage

import (
	"context"
	"encoding/base64"
	"io"

	"cloud.google.com/go/iam"
	"cloud.google.com/go/storage"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util"
)

func uploadToGCS(client *storage.Client, bucketName, objectName, data string) error {
	wc := client.Bucket(bucketName).Object(objectName).NewWriter(context.Background())
	b, _ := base64.StdEncoding.DecodeString(util.TrimBase64Mime(data))
	if _, err := io.Writer.Write(wc, b); err != nil {
		return err
	}
	return wc.Close()
}

// Check if an object in GCS is public or not
// Refer to https://stackoverflow.com/questions/68722565/how-to-check-if-a-file-in-gcp-storage-is-public-or-not
func isObjectPublic(client *storage.Client, bucketName, objectName string) (bool, error) {
	ctx := context.Background()
	bucket := client.Bucket(bucketName)
	attrs, err := bucket.Attrs(ctx)
	if err != nil {
		return false, err
	}

	public := false
	// When uniform bucket-level access is enabled on a bucket, Access Control Lists (ACLs) are disabled,
	// and only bucket-level Identity and Access Management (IAM) permissions grant access to that bucket and the objects it contains.
	// You revoke all access granted by object ACLs and the ability to administrate permissions using bucket ACLs.
	if attrs.UniformBucketLevelAccess.Enabled {
		policy, err := bucket.IAM().Policy(ctx)
		if err != nil {
			return false, err
		}
		for _, r := range policy.Roles() {
			for _, m := range policy.Members(r) {
				if m == iam.AllUsers {
					public = true
					break
				}
			}
		}
	} else {
		objAttrs, err := bucket.Object(objectName).Attrs(ctx)
		if err != nil {
			return false, err
		}
		for _, v := range objAttrs.ACL {
			if v.Entity == storage.AllUsers {
				public = true
				break
			}
		}
	}
	return public, nil
}
