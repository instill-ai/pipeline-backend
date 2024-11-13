//go:generate compogen readme ./config ./README.mdx

package freshdesk

import (
	"context"
	"fmt"
	"strings"
	"sync"

	_ "embed"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

const (
	version = "v2"

	taskGetTicket           = "TASK_GET_TICKET"
	taskCreateTicket        = "TASK_CREATE_TICKET"
	taskReplyToTicket       = "TASK_REPLY_TO_TICKET"
	taskCreateTicketNote    = "TASK_CREATE_TICKET_NOTE"
	taskGetContact          = "TASK_GET_CONTACT"
	taskCreateContact       = "TASK_CREATE_CONTACT"
	taskGetCompany          = "TASK_GET_COMPANY"
	taskCreateCompany       = "TASK_CREATE_COMPANY"
	taskGetAll              = "TASK_GET_ALL"
	taskGetAllConversations = "TASK_GET_ALL_CONVERSATIONS"
	taskGetProduct          = "TASK_GET_PRODUCT"
	taskGetAgent            = "TASK_GET_AGENT"
	taskGetRole             = "TASK_GET_ROLE"
	taskGetGroup            = "TASK_GET_GROUP"
	taskGetSkill            = "TASK_GET_SKILL"
)

var (
	//go:embed config/definition.json
	definitionJSON []byte
	//go:embed config/tasks.json
	tasksJSON []byte
	//go:embed config/setup.json
	setupJSON []byte

	once sync.Once
	comp *component
)

type component struct {
	base.Component
}

type execution struct {
	base.ComponentExecution
	client  FreshdeskInterface
	execute func(*structpb.Struct) (*structpb.Struct, error)
}

// Init returns an implementation of IComponent that implements the greeting
// task.
func Init(bc base.Component) *component {
	once.Do(func() {
		comp = &component{Component: bc}
		err := comp.LoadDefinition(definitionJSON, setupJSON, tasksJSON, nil, nil)
		if err != nil {
			panic(err)
		}
	})
	return comp
}

func (c *component) CreateExecution(x base.ComponentExecution) (base.IExecution, error) {
	e := &execution{
		ComponentExecution: x,
		client:             newClient(x.Setup, c.GetLogger()),
	}

	switch x.Task {
	case taskGetTicket:
		e.execute = e.TaskGetTicket
	case taskCreateTicket:
		e.execute = e.TaskCreateTicket
	case taskReplyToTicket:
		e.execute = e.TaskReplyToTicket
	case taskCreateTicketNote:
		e.execute = e.TaskCreateTicketNote
	case taskGetAllConversations:
		e.execute = e.TaskGetAllConversations
	case taskGetContact:
		e.execute = e.TaskGetContact
	case taskCreateContact:
		e.execute = e.TaskCreateContact
	case taskGetCompany:
		e.execute = e.TaskGetCompany
	case taskCreateCompany:
		e.execute = e.TaskCreateCompany
	case taskGetAll:
		e.execute = e.TaskGetAll
	case taskGetProduct:
		e.execute = e.TaskGetProduct
	case taskGetAgent:
		e.execute = e.TaskGetAgent
	case taskGetRole:
		e.execute = e.TaskGetRole
	case taskGetGroup:
		e.execute = e.TaskGetGroup
	case taskGetSkill:
		e.execute = e.TaskGetSkill
	default:
		return nil, fmt.Errorf("unsupported task")
	}

	return e, nil
}

func (e *execution) Execute(ctx context.Context, jobs []*base.Job) error {
	return base.SequentialExecutor(ctx, jobs, e.execute)
}

func convertTimestampResp(timestamp string) string {
	// freshdesk response timestamp is always in the format of YYYY-MM-DDTHH:MM:SSZ and in UTC.
	// this function will convert it to YYYY-MM-DD HH:MM:SS UTC

	if timestamp == "" {
		return timestamp
	}
	formattedTime := strings.Replace(timestamp, "T", " ", 1)
	formattedTime = strings.Replace(formattedTime, "Z", " ", 1)
	formattedTime += "UTC"

	return formattedTime
}

func checkForNilString(input *[]string) *[]string {
	if *input == nil {
		return &[]string{}
	}
	return input
}

func convertLanguageToCode(language string) string {

	switch language {
	case "Arabic":
		return "ar"
	case "Bosnian":
		return "bs"
	case "Bulgarian":
		return "bg"
	case "Catalan":
		return "ca"
	case "Chinese":
		return "zh-CN"
	case "Chinese (Traditional)":
		return "zh-TW"
	case "Croatian":
		return "hr"
	case "Czech":
		return "cs"
	case "Danish":
		return "da"
	case "Dutch":
		return "nl"
	case "English":
		return "en"
	case "Estonian":
		return "et"
	case "Filipino":
		return "fil"
	case "Finnish":
		return "fi"
	case "French":
		return "fr"
	case "German":
		return "de"
	case "Greek":
		return "el"
	case "Hebrew":
		return "he"
	case "Hungarian":
		return "hu"
	case "Icelandic":
		return "is"
	case "Indonesian":
		return "id"
	case "Italian":
		return "it"
	case "Japanese":
		return "ja-JP"
	case "Korean":
		return "ko"
	case "Latvian":
		return "lv-LV"
	case "Lithuanian":
		return "lt"
	case "Malay":
		return "ms"
	case "Norwegian":
		return "nb-NO"
	case "Polish":
		return "pl"
	case "Portuguese (BR)":
		return "pt-BR"
	case "Portuguese/Portugal":
		return "pt-PT"
	case "Romanian":
		return "ro"
	case "Russian":
		return "ru-RU"
	case "Serbian":
		return "sr"
	case "Slovak":
		return "sk"
	case "Slovenian":
		return "sl"
	case "Spanish":
		return "es"
	case "Spanish (Latin America)":
		return "es-LA"
	case "Swedish":
		return "sv-SE"
	case "Thai":
		return "th"
	case "Turkish":
		return "tr"
	case "Ukrainian":
		return "uk"
	case "Vietnamese":
		return "vi"
	default:
		return ""
	}
}

func convertCodeToLanguage(code string) string {
	switch code {
	case "en":
		return "English"
	case "ar":
		return "Arabic"
	case "bs":
		return "Bosnian"
	case "bg":
		return "Bulgarian"
	case "ca":
		return "Catalan"
	case "zh-CN":
		return "Chinese"
	case "zh-TW":
		return "Chinese (Traditional)"
	case "hr":
		return "Croatian"
	case "cs":
		return "Czech"
	case "da":
		return "Danish"
	case "nl":
		return "Dutch"
	case "et":
		return "Estonian"
	case "fil":
		return "Filipino"
	case "fi":
		return "Finnish"
	case "fr":
		return "French"
	case "de":
		return "German"
	case "el":
		return "Greek"
	case "he":
		return "Hebrew"
	case "hu":
		return "Hungarian"
	case "is":
		return "Icelandic"
	case "id":
		return "Indonesian"
	case "it":
		return "Italian"
	case "ja-JP":
		return "Japanese"
	case "ko":
		return "Korean"
	case "lv-LV":
		return "Latvian"
	case "lt":
		return "Lithuanian"
	case "ms":
		return "Malay"
	case "nb-NO":
		return "Norwegian"
	case "pl":
		return "Polish"
	case "pt-BR":
		return "Portuguese (BR)"
	case "pt-PT":
		return "Portuguese/Portugal"
	case "ro":
		return "Romanian"
	case "ru-RU":
		return "Russian"
	case "sr":
		return "Serbian"
	case "sk":
		return "Slovak"
	case "sl":
		return "Slovenian"
	case "es":
		return "Spanish"
	case "es-LA":
		return "Spanish (Latin America)"
	case "sv-SE":
		return "Swedish"
	case "th":
		return "Thai"
	case "tr":
		return "Turkish"
	case "uk":
		return "Ukrainian"
	case "vi":
		return "Vietnamese"
	default:
		return code
	}
}
