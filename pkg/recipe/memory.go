package recipe

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
)

const (
	SegMemory    = "memory"
	SegVariable  = "variable"
	SegSecret    = "secret"
	SegRecipe    = "recipe"
	SegOwner     = "owner_permalink"
	SegComponent = "component"
	SegIteration = "iterations"

	redisKeyPrefix = "pipeline_trigger"
)

// Key formats

// For regular components:
// pipeline_trigger:<workflowID>:recipe
// pipeline_trigger:<workflowID>:<batchIdx>:variable
// pipeline_trigger:<workflowID>:<batchIdx>:secret
// pipeline_trigger:<workflowID>:<batchIdx>:components:<compID>

// For the child pipeline in the iterator:
// pipeline_trigger:<workflowID>:components:<compID>:recipe
// pipeline_trigger:<workflowID>:components:<compID>:iterations:<iter>:components:<iterCompID>

type Memory struct {
	Variable  VariableMemory              `json:"variable"`
	Secret    SecretMemory                `json:"secret"`
	Component map[string]*ComponentMemory `json:"component"`
}

type VariableMemory map[string]any
type SecretMemory map[string]string
type ComponentMemory struct {
	Input   *ComponentIO     `json:"input"`
	Output  *ComponentIO     `json:"output"`
	Element any              `json:"element"` // for iterator
	Status  *ComponentStatus `json:"status"`
}

type BatchMemoryKey struct {
	Components     []map[string]string
	Secrets        []string
	Variables      []string
	Recipe         string
	OwnerPermalink string
}

type ComponentIO map[string]any

type ComponentStatus struct {
	Started   bool `json:"started"`
	Completed bool `json:"completed"`
	Skipped   bool `json:"skipped"`
}

func Write(ctx context.Context, rc *redis.Client, triggerID string, recipe *datamodel.Recipe, batchMemory []*Memory, ownerPermalink string) (*BatchMemoryKey, error) {

	batchSize := len(batchMemory)
	varKeys := make([]string, batchSize)
	secretKeys := make([]string, batchSize)
	for i := 0; i < batchSize; i++ {
		varKeys[i] = fmt.Sprintf("%s:%d:%s", triggerID, i, SegVariable)
		secretKeys[i] = fmt.Sprintf("%s:%d:%s", triggerID, i, SegSecret)
	}
	triggerStorageKey := &BatchMemoryKey{
		Recipe:         fmt.Sprintf("%s:%s", triggerID, SegRecipe),
		OwnerPermalink: fmt.Sprintf("%s:%s", triggerID, SegOwner),
		Secrets:        secretKeys,
		Variables:      varKeys,
		Components:     []map[string]string{},
	}

	var b []byte
	var err error

	if err := writeData(ctx, rc, triggerStorageKey.OwnerPermalink, ownerPermalink); err != nil {
		return nil, err
	}

	if recipe != nil {
		b, err = json.Marshal(recipe)
		if err != nil {
			return nil, err
		}
		if err := writeData(ctx, rc, triggerStorageKey.Recipe, b); err != nil {
			return nil, err
		}
	}

	for idx, memory := range batchMemory {
		if memory.Secret == nil {
			memory.Secret = map[string]string{}
		}

		b, err = json.Marshal(memory.Secret)
		if err != nil {
			return nil, err
		}
		if err := writeData(ctx, rc, triggerStorageKey.Secrets[idx], b); err != nil {
			return nil, err
		}

		if memory.Variable == nil {
			memory.Variable = map[string]any{}
		}
		b, err = json.Marshal(memory.Variable)
		if err != nil {
			return nil, err
		}
		if err := writeData(ctx, rc, triggerStorageKey.Variables[idx], b); err != nil {
			return nil, err
		}
	}

	triggerStorageKey.Components = make([]map[string]string, batchSize)

	return triggerStorageKey, nil
}

func WriteRecipe(ctx context.Context, rc *redis.Client, key string, recipe *datamodel.Recipe) error {
	var b []byte
	var err error

	if recipe != nil {
		b, err = json.Marshal(recipe)
		if err != nil {
			return err
		}
		if err := writeData(ctx, rc, key, b); err != nil {
			return err
		}

	}

	return nil
}

func LoadMemory(
	ctx context.Context,
	rc *redis.Client,
	key *BatchMemoryKey,
) ([]*Memory, error) {

	batchSize := len(key.Variables)
	memory := make([]*Memory, batchSize)

	for idx := range batchSize {
		memory[idx] = &Memory{
			Variable:  make(VariableMemory),
			Secret:    make(SecretMemory),
			Component: make(map[string]*ComponentMemory),
		}
		if err := loadData(ctx, rc, key.Variables[idx], &memory[idx].Variable); err != nil {
			return nil, err
		}
		if err := loadData(ctx, rc, key.Secrets[idx], &memory[idx].Secret); err != nil {
			return nil, err
		}
		for compID := range key.Components[idx] {
			m := ComponentMemory{}
			if err := loadData(ctx, rc, key.Components[idx][compID], &m); err != nil {
				return nil, err
			}
			memory[idx].Component[compID] = &m
		}
	}

	return memory, nil
}

func LoadMemoryByTriggerID(ctx context.Context, rc *redis.Client, triggerID string) ([]*Memory, error) {
	batchSize := getBatchSize(ctx, rc, triggerID)
	compIDs := getCompIDs(ctx, rc, triggerID)
	varKeys := make([]string, batchSize)
	compKeys := make([]map[string]string, batchSize)
	secretKeys := make([]string, batchSize)

	for i := range batchSize {
		varKeys[i] = fmt.Sprintf("%s:%d:%s", triggerID, i, SegVariable)
		secretKeys[i] = fmt.Sprintf("%s:%d:%s", triggerID, i, SegSecret)
		compKeys[i] = map[string]string{}
		for _, compID := range compIDs {
			compKeys[i][compID] = fmt.Sprintf("%s:%d:%s:%s", triggerID, i, SegComponent, compID)
		}
	}

	return LoadMemory(
		ctx,
		rc,
		&BatchMemoryKey{
			Variables:  varKeys,
			Secrets:    secretKeys,
			Components: compKeys,
		},
	)
}

func LoadRecipe(ctx context.Context, rc *redis.Client, key string) (*datamodel.Recipe, error) {
	recipe := &datamodel.Recipe{}

	if err := loadData(ctx, rc, key, recipe); err != nil {
		return nil, err
	}

	return recipe, nil
}

func LoadOwnerPermalink(ctx context.Context, rc *redis.Client, key string) string {
	return rc.Get(ctx, fmt.Sprintf("%s:%s:%s", redisKeyPrefix, key, SegOwner)).Val()
}

func Purge(ctx context.Context, rc *redis.Client, pipelineTriggerID string) {
	iter := rc.Scan(ctx, 0, fmt.Sprintf("%s:%s:*", redisKeyPrefix, pipelineTriggerID), 0).Iterator()
	for iter.Next(ctx) {
		rc.Del(ctx, iter.Val())
	}
}

func WriteComponentMemory(ctx context.Context, rc *redis.Client, key string, compID string, compsMem []*ComponentMemory) error {

	for idx, compMem := range compsMem {
		b, err := json.Marshal(compMem)
		if err != nil {
			return err
		}
		if err := writeData(ctx, rc, fmt.Sprintf("%s:%d:%s:%s", key, idx, SegComponent, compID), b); err != nil {
			return err
		}
	}

	return nil
}

func writeData(ctx context.Context, rc *redis.Client, key string, data any) error {
	if cmd := rc.Set(
		ctx,
		fmt.Sprintf("%s:%s", redisKeyPrefix, key),
		data,
		time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout)*time.Second,
	); cmd.Err() != nil {
		return cmd.Err()
	}
	return nil
}

func loadData(ctx context.Context, rc *redis.Client, key string, target any) error {
	b, err := rc.Get(ctx, fmt.Sprintf("%s:%s", redisKeyPrefix, key)).Bytes()
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, target)
	if err != nil {
		return err
	}
	return nil
}

func getBatchSize(ctx context.Context, rc *redis.Client, key string) int {
	iter := rc.Scan(ctx, 0, fmt.Sprintf("%s:%s:*:%s", redisKeyPrefix, key, SegVariable), 0).Iterator()
	batchSize := 0
	for iter.Next(ctx) {
		batchSize += 1
	}
	return batchSize
}

func getCompIDs(ctx context.Context, rc *redis.Client, key string) []string {
	iter := rc.Scan(ctx, 0, fmt.Sprintf("%s:%s:*:%s:*", redisKeyPrefix, key, SegComponent), 0).Iterator()
	compIDMap := map[string]bool{}
	for iter.Next(ctx) {
		key := iter.Val()
		keySplits := strings.Split(key, ":")
		compID := keySplits[4]
		compIDMap[compID] = true
	}
	compIDs := []string{}
	for k := range compIDMap {
		compIDs = append(compIDs, k)
	}
	return compIDs
}
