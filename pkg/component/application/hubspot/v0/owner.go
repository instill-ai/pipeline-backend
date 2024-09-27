package hubspot

import (
	"fmt"
	"strings"

	_ "embed"

	"google.golang.org/protobuf/types/known/structpb"

	hubspot "github.com/belong-inc/go-hubspot"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

// following go-hubspot sdk format

// API function for Owner
// API documentation: https://developers.hubspot.com/docs/api/crm/owners

type OwnerService interface {
	Get(ownerInfo string, infoType string) (*TaskGetOwnerResp, error)
}

type OwnerServiceOp struct {
	client    *hubspot.Client
	ownerPath string
}

func (s *OwnerServiceOp) Get(ownerInfo string, infoType string) (*TaskGetOwnerResp, error) {
	resource := &TaskGetOwnerResp{}
	if err := s.client.Get(s.ownerPath+"/"+ownerInfo+"/?idProperty="+infoType, resource, nil); err != nil {
		return nil, err
	}

	return resource, nil
}

// Get Owner

type TaskGetOwnerInputstruct struct {
	IDType string `json:"id-type"`
	ID     string `json:"id"`
}

type TaskGetOwnerResp struct {
	FirstName string                 `json:"firstName"`
	LastName  string                 `json:"lastName"`
	Email     string                 `json:"email"`
	OwnerID   string                 `json:"id"`
	UserID    int                    `json:"userId"`
	Teams     []taskGetOwnerRespTeam `json:"teams,omitempty"`

	CreatedAt *hubspot.HsTime `json:"createdAt"`
	UpdatedAt *hubspot.HsTime `json:"updatedAt"`
	Archived  bool            `json:"archived"`
}

type taskGetOwnerRespTeam struct {
	Name    string `json:"name"`
	ID      string `json:"id"`
	Primary bool   `json:"primary"`
}

type TaskGetOwnerOutput struct {
	FirstName string                   `json:"first-name"`
	LastName  string                   `json:"last-name"`
	Email     string                   `json:"email"`
	OwnerID   string                   `json:"owner-id"`
	UserID    string                   `json:"user-id"` //UserID can be string in other hubspot schema. I will stick to string as well so that it is consistent with other ID types
	Teams     []taskGetOwnerOutputTeam `json:"teams,omitempty"`

	CreatedAt string `json:"created-at"`
	UpdatedAt string `json:"updated-at"`
	Archived  bool   `json:"archived"`
}

type taskGetOwnerOutputTeam struct {
	Name    string `json:"team-name"`
	ID      string `json:"team-id"`
	Primary bool   `json:"team-primary"`
}

func (e *execution) GetOwner(input *structpb.Struct) (*structpb.Struct, error) {

	inputStruct := TaskGetOwnerInputstruct{}

	err := base.ConvertFromStructpb(input, &inputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert input to struct: %v", err)
	}

	var infoType string
	switch inputStruct.IDType {
	case "Owner ID":
		infoType = "id"
	case "User ID":
		infoType = "userId"
	}

	res, err := e.client.Owner.Get(inputStruct.ID, infoType)

	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return nil, fmt.Errorf("404: unable to read response from hubspot: no owner was found")
		} else {
			return nil, err
		}
	}

	outputStruct := TaskGetOwnerOutput{
		FirstName: res.FirstName,
		LastName:  res.LastName,
		Email:     res.Email,
		OwnerID:   res.OwnerID,
		UserID:    fmt.Sprint(res.UserID),
		CreatedAt: res.CreatedAt.String(),
		UpdatedAt: res.UpdatedAt.String(),
		Archived:  res.Archived,
	}

	// convert to output struct
	for _, resTeam := range res.Teams {
		outputTeam := taskGetOwnerOutputTeam(resTeam)
		outputStruct.Teams = append(outputStruct.Teams, outputTeam)
	}

	output, err := base.ConvertToStructpb(outputStruct)
	if err != nil {
		return nil, fmt.Errorf("failed to convert output to struct: %v", err)
	}

	return output, nil
}
