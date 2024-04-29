package recipe

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
)

const (
	MemoryKey  = "memory"
	TriggerKey = "trigger"
	SecretsKey = "secrets"
)

type TriggerMemory struct {
	Components      map[string]ComponentsMemory `json:"components"`
	Inputs          []InputsMemory              `json:"inputs"`
	Secrets         map[string]string           `json:"secrets"`
	Vars            map[string]any              `json:"vars"`
	SystemVariables SystemVariables             `json:"sys_vars"`
}

type ComponentsMemory []*ComponentItemMemory
type InputsMemory map[string]any

type ComponentItemMemory struct {
	Input   *ComponentIO     `json:"input"`
	Output  *ComponentIO     `json:"output"`
	Element any              `json:"element"` // for iterator
	Status  *ComponentStatus `json:"status"`
}

type ComponentIO map[string]any

type ComponentStatus struct {
	Started   bool `json:"started"`
	Completed bool `json:"completed"`
	Skipped   bool `json:"skipped"`
}

func WriteMemoryAndRecipe(ctx context.Context, rc *redis.Client, redisKey string, recipe *datamodel.Recipe, memory *TriggerMemory, ownerPermalink string) error {
	var b []byte
	var err error

	rc.Set(
		ctx,
		redisKey+":owner_permalink",
		ownerPermalink,
		time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout)*time.Second,
	)

	if recipe != nil {
		b, err = json.Marshal(recipe)
		if err != nil {
			return err
		}
		rc.Set(
			ctx,
			redisKey+":recipe",
			b,
			time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout)*time.Second,
		)
	}

	if memory.Secrets != nil {
		b, err = json.Marshal(memory.Secrets)
		if err != nil {
			return err
		}
		rc.Set(
			ctx,
			redisKey+":secrets",
			b,
			time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout)*time.Second,
		)
	}

	if memory.Vars != nil {
		b, err = json.Marshal(memory.Vars)
		if err != nil {
			return err
		}
		rc.Set(
			ctx,
			redisKey+":vars",
			b,
			time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout)*time.Second,
		)
	}

	if memory.Inputs != nil {
		b, err = json.Marshal(memory.Inputs)
		if err != nil {
			return err
		}
		rc.Set(
			ctx,
			redisKey+":inputs",
			b,
			time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout)*time.Second,
		)
	}

	for compID, compMem := range memory.Components {
		b, err = json.Marshal(compMem)
		if err != nil {
			return err
		}
		rc.Set(
			ctx,
			redisKey+":components:"+compID,
			b,
			time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout)*time.Second,
		)
	}

	return nil
}

func LoadMemory(ctx context.Context, rc *redis.Client, redisKey string) (*TriggerMemory, error) {
	memory := TriggerMemory{
		Secrets:    map[string]string{},
		Vars:       map[string]any{},
		Inputs:     []InputsMemory{},
		Components: map[string]ComponentsMemory{},
	}

	if b, err := rc.Get(ctx, redisKey+":secrets").Bytes(); err == nil {
		err = json.Unmarshal(b, &memory.Secrets)
		if err != nil {
			return nil, err
		}
	}

	if b, err := rc.Get(ctx, redisKey+":vars").Bytes(); err == nil {
		err = json.Unmarshal(b, &memory.Vars)
		if err != nil {
			return nil, err
		}
	}

	if b, err := rc.Get(ctx, redisKey+":inputs").Bytes(); err == nil {
		err = json.Unmarshal(b, &memory.Inputs)
		if err != nil {
			return nil, err
		}
	}

	iter := rc.Scan(ctx, 0, redisKey+":components:*", 0).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		var result []*ComponentItemMemory
		b, err := rc.Get(ctx, key).Bytes()
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(b, &result)
		if err != nil {
			return nil, err
		}
		memory.Components[strings.Split(key, ":")[len(strings.Split(key, ":"))-1]] = result

	}

	return &memory, nil
}

func LoadRecipe(ctx context.Context, rc *redis.Client, redisKey string) (*datamodel.Recipe, error) {
	recipe := &datamodel.Recipe{}

	var b []byte

	b, err := rc.Get(ctx, redisKey+":recipe").Bytes()
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(b, recipe)
	if err != nil {
		return nil, err
	}

	return recipe, nil
}

func LoadOwnerPermalink(ctx context.Context, rc *redis.Client, redisKey string) string {
	return rc.Get(ctx, redisKey+":owner_permalink").Val()
}

func PurgeMemory(ctx context.Context, rc *redis.Client, redisKey string) {
	iter := rc.Scan(ctx, 0, redisKey+":*", 0).Iterator()
	for iter.Next(ctx) {
		rc.Del(ctx, iter.Val())
	}
}

func WriteComponentMemory(ctx context.Context, rc *redis.Client, redisKey string, compID string, compsMem []*ComponentItemMemory) error {

	b, err := json.Marshal(compsMem)
	if err != nil {
		return err
	}
	rc.Set(
		ctx,
		redisKey+":components:"+compID,
		b,
		time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout)*time.Second,
	)
	return nil
}
