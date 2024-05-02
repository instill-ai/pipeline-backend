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
	SegTrigger   = "trigger"
	SegSecrets   = "secrets"
	SegRecipe    = "recipe"
	SegOwner     = "owner_permalink"
	SegVars      = "vars"
	SegComponent = "components"
	SegIteration = "iterations"
	SegInputs    = "inputs"

	redisKeyPrefix = "pipeline_trigger"
)

// Key formats

// For regular components:
// pipeline_trigger:<workflowID>:recipe
// pipeline_trigger:<workflowID>:vars
// pipeline_trigger:<workflowID>:secrets
// pipeline_trigger:<workflowID>:inputs:<batchIdx>
// pipeline_trigger:<workflowID>:components:<compID>:<batchIdx>

// For the child pipeline in the iterator:
// pipeline_trigger:<workflowID>:components:<compID>:recipe
// pipeline_trigger:<workflowID>:components:<compID>:iterations:<iter>:components:<iterCompID>

type TriggerMemory struct {
	Components      map[string]ComponentsMemory `json:"components"`
	Inputs          []InputsMemory              `json:"inputs"`
	Secrets         map[string]string           `json:"secrets"`
	Vars            map[string]any              `json:"vars"`
	SystemVariables SystemVariables             `json:"sys_vars"`
}

type TriggerMemoryKey struct {
	Components     map[string][]string
	Inputs         []string
	Secrets        string
	Vars           string
	Recipe         string
	OwnerPermalink string
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

func Write(ctx context.Context, rc *redis.Client, triggerID string, recipe *datamodel.Recipe, memory *TriggerMemory, ownerPermalink string) (*TriggerMemoryKey, error) {

	batchSize := len(memory.Inputs)
	inputStorageKeys := make([]string, batchSize)
	for i := 0; i < batchSize; i++ {
		inputStorageKeys[i] = fmt.Sprintf("%s:%s:%d", triggerID, SegInputs, i)
	}
	triggerStorageKey := &TriggerMemoryKey{
		Recipe:         fmt.Sprintf("%s:%s", triggerID, SegRecipe),
		OwnerPermalink: fmt.Sprintf("%s:%s", triggerID, SegOwner),
		Secrets:        fmt.Sprintf("%s:%s", triggerID, SegSecrets),
		Vars:           fmt.Sprintf("%s:%s", triggerID, SegVars),
		Inputs:         inputStorageKeys,
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

	if memory.Secrets == nil {
		memory.Secrets = map[string]string{}
	}

	b, err = json.Marshal(memory.Secrets)
	if err != nil {
		return nil, err
	}
	if err := writeData(ctx, rc, triggerStorageKey.Secrets, b); err != nil {
		return nil, err
	}

	if memory.Vars == nil {
		memory.Vars = map[string]any{}
	}

	b, err = json.Marshal(memory.Vars)
	if err != nil {
		return nil, err
	}
	if err := writeData(ctx, rc, triggerStorageKey.Vars, b); err != nil {
		return nil, err
	}

	for idx, input := range memory.Inputs {
		b, err = json.Marshal(input)
		if err != nil {
			return nil, err
		}
		if err := writeData(ctx, rc, triggerStorageKey.Inputs[idx], b); err != nil {
			return nil, err
		}
	}

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
	key *TriggerMemoryKey,
) (*TriggerMemory, error) {
	memory := &TriggerMemory{
		Secrets:    map[string]string{},
		Vars:       map[string]any{},
		Inputs:     []InputsMemory{},
		Components: map[string]ComponentsMemory{},
	}

	if err := loadData(ctx, rc, key.Secrets, &memory.Secrets); err != nil {
		return nil, err
	}

	if err := loadData(ctx, rc, key.Vars, &memory.Vars); err != nil {
		return nil, err
	}

	memory.Inputs = make([]InputsMemory, len(key.Inputs))
	cacheInputs := map[string]*InputsMemory{}
	for idx, k := range key.Inputs {
		// Note: In an iterator, the same key might exist. We can use a local map to reduce Redis requests.
		if v, ok := cacheInputs[k]; ok {
			memory.Inputs[idx] = *v
			continue
		}
		if err := loadData(ctx, rc, k, &memory.Inputs[idx]); err != nil {
			return nil, err
		}
		cacheInputs[k] = &memory.Inputs[idx]
	}

	for compID, ks := range key.Components {
		memory.Components[compID] = make(ComponentsMemory, len(ks))
		cacheComps := map[string]*ComponentItemMemory{}
		for idx, k := range ks {
			// Note: In an iterator, the same key might exist. We can use a local map to reduce Redis requests.
			if v, ok := cacheComps[k]; ok {
				memory.Components[compID][idx] = v
				continue
			}
			var result ComponentItemMemory
			if err := loadData(ctx, rc, k, &result); err != nil {
				return nil, err
			}
			memory.Components[compID][idx] = &result
			cacheComps[k] = &result
		}
	}
	return memory, nil
}

func LoadMemoryByTriggerID(ctx context.Context, rc *redis.Client, triggerID string) (*TriggerMemory, error) {
	batchSize := getBatchSize(ctx, rc, triggerID)
	compIDs := getCompIDs(ctx, rc, triggerID)
	keyInputs := make([]string, batchSize)
	for i := 0; i < batchSize; i++ {
		keyInputs[i] = fmt.Sprintf("%s:%s:%d", triggerID, SegInputs, i)
	}
	keyComps := map[string][]string{}
	for _, compID := range compIDs {
		keyComps[compID] = make([]string, batchSize)
		for i := 0; i < batchSize; i++ {
			keyComps[compID][i] = fmt.Sprintf("%s:%s:%s:%d", triggerID, SegComponent, compID, i)
		}
	}

	return LoadMemory(
		ctx,
		rc,
		&TriggerMemoryKey{
			Components: keyComps,
			Inputs:     keyInputs,
			Secrets:    fmt.Sprintf("%s:%s", triggerID, SegSecrets),
			Vars:       fmt.Sprintf("%s:%s", triggerID, SegVars),
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

func WriteComponentMemory(ctx context.Context, rc *redis.Client, key string, compsMem []*ComponentItemMemory) error {

	for idx, itemMem := range compsMem {
		b, err := json.Marshal(itemMem)
		if err != nil {
			return err
		}
		if err := writeData(ctx, rc, fmt.Sprintf("%s:%d", key, idx), b); err != nil {
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
	iter := rc.Scan(ctx, 0, fmt.Sprintf("%s:%s:%s:*", redisKeyPrefix, key, SegInputs), 0).Iterator()
	batchSize := 0
	for iter.Next(ctx) {
		batchSize += 1
	}
	return batchSize
}

func getCompIDs(ctx context.Context, rc *redis.Client, key string) []string {
	iter := rc.Scan(ctx, 0, fmt.Sprintf("%s:%s:%s:*", redisKeyPrefix, key, SegComponent), 0).Iterator()
	compIDMap := map[string]bool{}
	for iter.Next(ctx) {
		key := iter.Val()
		keySplits := strings.Split(key, ":")
		compID := keySplits[3]
		compIDMap[compID] = true
	}
	compIDs := []string{}
	for k := range compIDMap {
		compIDs = append(compIDs, k)
	}
	return compIDs
}
