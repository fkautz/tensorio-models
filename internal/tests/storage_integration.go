package tests

import (
	"context"
	"github.com/doc-ai/tensorio-models/storage"
	"github.com/stretchr/testify/assert"
	"log"
	"testing"
	"time"
)

func Test_AddModel(t *testing.T, store storage.RepositoryStorage) {
	ctx := context.Background()

	// ensure model doesn't exist
	_, err := store.GetModel(ctx, "add_model")
	assert.Error(t, err, "Model should not exist")

	model := storage.Model{
		ModelId:                  "add_model",
		Description:              "description of a model",
		CanonicalHyperParameters: "canonical hyper params",
	}

	// add model
	err = store.AddModel(ctx, model)
	assert.NoError(t, err)

	// get model and ensure it hasn't changed
	storedModel, err := store.GetModel(ctx, "add_model")
	assert.Equal(t, model, storedModel)
	assert.NoError(t, err)

	model2 := storage.Model{
		ModelId:                  "add_model2",
		Description:              "description of a model",
		CanonicalHyperParameters: "canonical hyper params",
	}

	err = store.AddModel(ctx, model2)
	assert.NoError(t, err)

	// see adding same models with conflicting name fails
	err = store.AddModel(ctx, model)
	assert.Error(t, err, "Model should fail when conflict on add")
	err = store.AddModel(ctx, model2)
	assert.Error(t, err, "Model should fail when conflict on add")

	storedModel, err = store.GetModel(ctx, "add_model")
	assert.Equal(t, model, storedModel)
	assert.NoError(t, err)

	storedModel2, err := store.GetModel(ctx, "add_model2")
	assert.Equal(t, model2, storedModel2)
	assert.NoError(t, err)
}

func Test_ListModels(t *testing.T, store storage.RepositoryStorage) {
	ctx := context.Background()

	// test empty case
	list, err := store.ListModels(ctx, "model1", 2)
	assert.Equal(t, []string{}, list)
	assert.NoError(t, err)

	model := storage.Model{
		ModelId:                  "model1",
		Description:              "description of a model",
		CanonicalHyperParameters: "canonical hyper params",
	}
	store.AddModel(ctx, model)

	model.ModelId = "model3"
	store.AddModel(ctx, model)

	model.ModelId = "model2"
	store.AddModel(ctx, model)

	model.ModelId = "model4"
	store.AddModel(ctx, model)

	list, err = store.ListModels(ctx, "nothing", 2)
	assert.Equal(t, []string{}, list)
	assert.NoError(t, err)

	list, err = store.ListModels(ctx, "model1", 2)
	assert.Equal(t, []string{"model1", "model2"}, list)
	assert.NoError(t, err)

	list, err = store.ListModels(ctx, "model1", 4)
	assert.Equal(t, []string{"model1", "model2", "model3", "model4"}, list)
	assert.NoError(t, err)

	list, err = store.ListModels(ctx, "model1", 5)
	assert.Equal(t, []string{"model1", "model2", "model3", "model4"}, list)
	assert.NoError(t, err)

	list, err = store.ListModels(ctx, "a", 2)
	assert.Equal(t, []string{"model1", "model2"}, list)
	assert.NoError(t, err)

	list, err = store.ListModels(ctx, "model2a", 2)
	assert.Equal(t, []string{"model3", "model4"}, list)
	assert.NoError(t, err)

	list, err = store.ListModels(ctx, "a", 0)
	assert.Equal(t, []string{}, list)
	assert.NoError(t, err)

	list, err = store.ListModels(ctx, "", 2)
	assert.Equal(t, []string{"model1", "model2"}, list)
	assert.NoError(t, err)
}

func Test_UpdateModels(t *testing.T, store storage.RepositoryStorage) {
	ctx := context.Background()

	// test no model exists
	model := storage.Model{
		ModelId:                  "model1",
		Description:              "old description",
		CanonicalHyperParameters: "old canonical",
	}
	_, err := store.UpdateModel(ctx, model)
	assert.Error(t, err, "Model should not exist")

	// add a model
	store.AddModel(ctx, model)

	// update model
	modelUpdate := storage.Model{
		ModelId:                  "model1",
		Description:              "",
		CanonicalHyperParameters: "",
	}
	updatedModel, err := store.UpdateModel(ctx, modelUpdate)

	// check if new model is updated, in this case empty
	assert.NoError(t, err)
	assert.Equal(t, model, updatedModel)

	modelUpdate = storage.Model{
		ModelId:                  "model1",
		Description:              "new description",
		CanonicalHyperParameters: "new canonical",
	}
	updatedModel, err = store.UpdateModel(ctx, modelUpdate)

	// check if new model is updated
	assert.NoError(t, err)
	assert.Equal(t, modelUpdate, updatedModel)
}

func Test_AddHyperParameters(t *testing.T, store storage.RepositoryStorage) {
	ctx := context.Background()

	_, err := store.GetHyperParameters(ctx, "model1", "param1")
	assert.Error(t, err, "Hyperparameter should not exist")

	model := storage.Model{
		ModelId:                  "model1",
		Description:              "description1",
		CanonicalHyperParameters: "canonical1",
	}
	store.AddModel(ctx, model)

	_, err = store.GetHyperParameters(ctx, "model1", "param1")
	assert.Error(t, err, "Hyperparameter should not exist")

	params := storage.HyperParameters{
		ModelId:             "fail",
		HyperParametersId:   "paramid",
		CanonicalCheckpoint: "checkpoint",
	}

	// error on add params to non existence model
	err = store.AddHyperParameters(ctx, params)
	assert.Error(t, err)

	// add params
	params.ModelId = "model1"
	err = store.AddHyperParameters(ctx, params)
	assert.NoError(t, err)

	// expect error on conflict
	err = store.AddHyperParameters(ctx, params)
	assert.Error(t, err)
}

func Test_ListHyperParams(t *testing.T, store storage.RepositoryStorage) {
	ctx := context.Background()

	_, err := store.ListHyperParameters(ctx, "nomodel", "marker", 2)
	assert.Error(t, err)

	model := storage.Model{
		ModelId:                  "model1",
		Description:              "description1",
		CanonicalHyperParameters: "canonical1",
	}
	store.AddModel(ctx, model)
	model.ModelId = "model2"
	store.AddModel(ctx, model)
	params, err := store.ListHyperParameters(ctx, "model1", "marker", 2)
	assert.Equal(t, []string{}, params)
	assert.NoError(t, err)

	param := storage.HyperParameters{
		ModelId:             "model1",
		HyperParametersId:   "param1",
		CanonicalCheckpoint: "canon1",
	}
	store.AddHyperParameters(ctx, param)

	param.HyperParametersId = "param3"
	store.AddHyperParameters(ctx, param)

	param.ModelId = "model2"
	param.HyperParametersId = "param1"
	store.AddHyperParameters(ctx, param)

	param.ModelId = "model1"
	param.HyperParametersId = "param2"
	store.AddHyperParameters(ctx, param)

	param.HyperParametersId = "param4"
	store.AddHyperParameters(ctx, param)

	param = storage.HyperParameters{
		ModelId:             "model1",
		HyperParametersId:   "param1",
		CanonicalCheckpoint: "canon1",
	}

	params, err = store.ListHyperParameters(ctx, "model1", "marker", 2)
	assert.Equal(t, []string{"model1:param1", "model1:param2"}, params)
	assert.NoError(t, err)

	params, err = store.ListHyperParameters(ctx, "model1", "marker", 5)
	assert.Equal(t, []string{"model1:param1", "model1:param2", "model1:param3", "model1:param4"}, params)
	assert.NoError(t, err)

	params, err = store.ListHyperParameters(ctx, "model1", "param22", 5)
	assert.Equal(t, []string{"model1:param3", "model1:param4"}, params)
	assert.NoError(t, err)

	params, err = store.ListHyperParameters(ctx, "model2", "", 5)
	assert.Equal(t, []string{"model2:param1"}, params)
	assert.NoError(t, err)
}

func Test_UpdateHyperParams(t *testing.T, store storage.RepositoryStorage) {
	ctx := context.Background()

	model := storage.Model{
		ModelId:                  "model1",
		Description:              "desc",
		CanonicalHyperParameters: "canon",
	}

	hyperParameters := storage.HyperParameters{
		ModelId:             "model1",
		HyperParametersId:   "param1",
		CanonicalCheckpoint: "checkpoint1",
		HyperParameters:     map[string]string{"hp1": "1"},
	}

	_, err := store.UpdateHyperParameters(ctx, hyperParameters)
	assert.Error(t, err)

	store.AddModel(ctx, model)
	_, err = store.UpdateHyperParameters(ctx, hyperParameters)
	assert.Error(t, err)

	store.AddHyperParameters(ctx, hyperParameters)

	hyperParametersUpdate := storage.HyperParameters{
		ModelId:             "model1",
		HyperParametersId:   "param1",
		CanonicalCheckpoint: "",
		HyperParameters:     nil,
	}

	updatedHyperParameters, err := store.UpdateHyperParameters(ctx, hyperParametersUpdate)
	assert.Equal(t, hyperParameters, updatedHyperParameters)

	hyperParametersUpdate.CanonicalCheckpoint = "checkpoint2"
	hyperParametersUpdate.HyperParameters = make(map[string]string)
	hyperParametersUpdate.HyperParameters["hp1"] = "1.1"
	hyperParametersUpdate.HyperParameters["hp2"] = "2"
	expectedHyperParameters := storage.HyperParameters{
		ModelId:             "model1",
		HyperParametersId:   "param1",
		CanonicalCheckpoint: "checkpoint2",
		HyperParameters:     map[string]string{"hp1": "1.1", "hp2": "2"},
	}
	updatedHyperParameters, err = store.UpdateHyperParameters(ctx, hyperParametersUpdate)
	assert.Equal(t, expectedHyperParameters, updatedHyperParameters)
}

func Test_AddCheckpoint(t *testing.T, store storage.RepositoryStorage) {
	ctx := context.Background()

	checkpoint1 := storage.Checkpoint{
		ModelId:           "model1",
		HyperParametersId: "params1",
		CheckpointId:      "cp1",
		Link:              "link1",
		CreatedAt:         time.Now(),
		Info:              map[string]string{"info1": "1"},
	}
	_, err := store.GetCheckpoint(ctx, "model1", "param1", "cp1")
	assert.Error(t, err)

	err = store.AddCheckpoint(ctx, checkpoint1)
	assert.Error(t, err)

	model1 := storage.Model{
		ModelId:                  "model1",
		Description:              "desc",
		CanonicalHyperParameters: "canon",
	}
	store.AddModel(ctx, model1)

	_, err = store.GetCheckpoint(ctx, "model1", "params1", "cp1")
	assert.Error(t, err)

	err = store.AddCheckpoint(ctx, checkpoint1)
	assert.Error(t, err)

	params1 := storage.HyperParameters{
		ModelId:             "model1",
		HyperParametersId:   "params1",
		CanonicalCheckpoint: "canon1",
		HyperParameters:     map[string]string{"hp1": "1"},
	}
	store.AddHyperParameters(ctx, params1)

	_, err = store.GetCheckpoint(ctx, "model1", "params1", "cp1")
	assert.Error(t, err)

	err = store.AddCheckpoint(ctx, checkpoint1)
	assert.NoError(t, err)

	err = store.AddCheckpoint(ctx, checkpoint1)
	assert.Error(t, err)

	addedCheckpoint, err := store.GetCheckpoint(ctx, "model1", "params1", "cp1")
	log.Println(addedCheckpoint)
	assert.Equal(t, checkpoint1, addedCheckpoint)
	assert.NoError(t, err)
}

func Test_ListCheckpoints(t *testing.T, store storage.RepositoryStorage) {
	ctx := context.Background()

	_, err := store.ListCheckpoints(ctx, "model1", "params1", "", 4)
	assert.Error(t, err)

	model1 := storage.Model{
		ModelId:                  "model1",
		Description:              "desc",
		CanonicalHyperParameters: "canon",
	}
	store.AddModel(ctx, model1)

	_, err = store.ListCheckpoints(ctx, "model1", "params1", "", 4)
	assert.Error(t, err)

	params1 := storage.HyperParameters{
		ModelId:             "model1",
		HyperParametersId:   "params1",
		CanonicalCheckpoint: "canon1",
		HyperParameters:     map[string]string{"hp1": "1"},
	}
	store.AddHyperParameters(ctx, params1)
	_, err = store.ListCheckpoints(ctx, "model1", "params1", "", 4)
	assert.NoError(t, err)

	checkpoint1 := storage.Checkpoint{
		ModelId:           "model1",
		HyperParametersId: "params1",
		CheckpointId:      "cp1",
		Link:              "link1",
		CreatedAt:         time.Now(),
		Info:              map[string]string{"info1": "1"},
	}
	store.AddCheckpoint(ctx, checkpoint1)
	checkpoints, err := store.ListCheckpoints(ctx, "model1", "params1", "", 4)
	assert.Equal(t, []string{"model1:params1:cp1"}, checkpoints)
	assert.NoError(t, err)

	checkpoint1.CheckpointId = "cp3"
	store.AddCheckpoint(ctx, checkpoint1)

	checkpoint1.CheckpointId = "cp2"
	store.AddCheckpoint(ctx, checkpoint1)

	checkpoint1.CheckpointId = "cp4"
	store.AddCheckpoint(ctx, checkpoint1)

	checkpoints, err = store.ListCheckpoints(ctx, "model1", "params1", "", 4)
	assert.Equal(t, []string{
		"model1:params1:cp1",
		"model1:params1:cp2",
		"model1:params1:cp3",
		"model1:params1:cp4",
	}, checkpoints)
	assert.NoError(t, err)

	checkpoints, err = store.ListCheckpoints(ctx, "model1", "params1", "cp22", 4)
	assert.Equal(t, []string{
		"model1:params1:cp3",
		"model1:params1:cp4",
	}, checkpoints)
	assert.NoError(t, err)
}
