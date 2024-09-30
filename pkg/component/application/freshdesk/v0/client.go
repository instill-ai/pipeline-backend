package freshdesk

import (
	"encoding/base64"
	"fmt"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util/httpclient"
)

func newClient(setup *structpb.Struct, logger *zap.Logger) *FreshdeskClient {
	basePath := fmt.Sprintf("https://%s.freshdesk.com/api", getDomain(setup))

	c := httpclient.New("Freshdesk", basePath+"/"+version,
		httpclient.WithLogger(logger),
		httpclient.WithEndUserError(new(errBody)),
	)

	c.Header.Set("Authorization", getAPIKey(setup))

	w := &FreshdeskClient{httpclient: c}

	return w
}

// sometimes it will give an array of errors, other times it wont.
type errBody struct {
	Description string `json:"description"`
	Errors      []struct {
		Field   string `json:"field"`
		Message string `json:"message"`
		Code    string `json:"code"`
	} `json:"errors"`
	ErrorMessage string `json:"message"`
}

func (e errBody) Message() string {
	var errReturn string
	for index, err := range e.Errors {
		if index > 0 {
			errReturn += ". "
		}

		errReturn += err.Message
		if err.Field != "" {
			errReturn += ", field: " + err.Field
		}
		if err.Code != "" {
			errReturn += ", code: " + err.Code
		}
	}

	errReturn += e.ErrorMessage

	return errReturn
}

func getAPIKey(setup *structpb.Struct) string {
	apiKey := setup.GetFields()["api-key"].GetStringValue()

	// In Freshdesk, the format is api-key:X. Afterward, it needs to be encoded in base64
	encodedKey := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:X", apiKey)))
	return encodedKey
}

func getDomain(setup *structpb.Struct) string {
	return setup.GetFields()["domain"].GetStringValue()
}

type FreshdeskClient struct {
	httpclient *httpclient.Client
}

type FreshdeskInterface interface {
	GetTicket(ticketID int64) (*TaskGetTicketResponse, error)
	CreateTicket(req *TaskCreateTicketReq) (*TaskCreateTicketResponse, error)
	ReplyToTicket(ticketID int64, req *TaskReplyToTicketReq) (*TaskReplyToTicketResponse, error)
	CreateTicketNote(ticketID int64, req *TaskCreateTicketNoteReq) (*TaskCreateTicketNoteResponse, error)
	GetContact(contactID int64) (*TaskGetContactResponse, error)
	CreateContact(req *TaskCreateContactReq) (*TaskCreateContactResponse, error)
	GetCompany(companyID int64) (*TaskGetCompanyResponse, error)
	CreateCompany(req *TaskCreateCompanyReq) (*TaskCreateCompanyResponse, error)
	GetAll(objectType string, pagination bool, paginationPath string) ([]TaskGetAllResponse, string, error)
	GetAllConversations(ticketID int64, pagination bool, paginationPath string) ([]TaskGetAllConversationsResponse, string, error)
	GetProduct(productID int64) (*TaskGetProductResponse, error)
	GetAgent(agentID int64) (*TaskGetAgentResponse, error)
	GetRole(roleID int64) (*TaskGetRoleResponse, error)
	GetGroup(groupID int64) (*TaskGetGroupResponse, error)
	GetSkill(skillID int64) (*TaskGetSkillResponse, error)
}
