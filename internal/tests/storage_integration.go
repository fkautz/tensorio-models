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
		Details:                  "details of a model",
		CanonicalHyperparameters: "canonical hyper params",
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
		Details:                  "details of a model",
		CanonicalHyperparameters: "canonical hyper params",
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
		Details:                  "details of a model",
		CanonicalHyperparameters: "canonical hyper params",
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
		Details:                  "old detail",
		CanonicalHyperparameters: "old canonical",
	}
	_, err := store.UpdateModel(ctx, model)
	assert.Error(t, err, "Model should not exist")

	// add a model
	store.AddModel(ctx, model)

	// update model
	modelUpdate := storage.Model{
		ModelId:                  "model1",
		Details:                  "",
		CanonicalHyperparameters: "",
	}
	updatedModel, err := store.UpdateModel(ctx, modelUpdate)

	// check if new model is updated, in this case empty
	assert.NoError(t, err)
	assert.Equal(t, model, updatedModel)

	modelUpdate = storage.Model{
		ModelId:                  "model1",
		Details:                  "new detail",
		CanonicalHyperparameters: "new canonical",
	}
	updatedModel, err = store.UpdateModel(ctx, modelUpdate)

	// check if new model is updated
	assert.NoError(t, err)
	assert.Equal(t, modelUpdate, updatedModel)
}

func Test_AddHyperparameters(t *testing.T, store storage.RepositoryStorage) {
	ctx := context.Background()

	_, err := store.GetHyperparameters(ctx, "model1", "param1")
	assert.Error(t, err, "Hyperparameter should not exist")

	model := storage.Model{
		ModelId:                  "model1",
		Details:                  "detail1",
		CanonicalHyperparameters: "canonical1",
	}
	store.AddModel(ctx, model)

	_, err = store.GetHyperparameters(ctx, "model1", "param1")
	assert.Error(t, err, "Hyperparameter should not exist")

	params := storage.Hyperparameters{
		ModelId:             "fail",
		HyperparametersId:   "paramid",
		CanonicalCheckpoint: "checkpoint",
	}

	// error on add params to non existence model
	err = store.AddHyperparameters(ctx, params)
	assert.Error(t, err)

	// add params
	params.ModelId = "model1"
	err = store.AddHyperparameters(ctx, params)
	assert.NoError(t, err)

	// expect error on conflict
	err = store.AddHyperparameters(ctx, params)
	assert.Error(t, err)
}

func Test_ListHyperparams(t *testing.T, store storage.RepositoryStorage) {
	ctx := context.Background()

	_, err := store.ListHyperparameters(ctx, "nomodel", "marker", 2)
	assert.Error(t, err)

	model := storage.Model{
		ModelId:                  "model1",
		Details:                  "detail1",
		CanonicalHyperparameters: "canonical1",
	}
	store.AddModel(ctx, model)
	model.ModelId = "model2"
	store.AddModel(ctx, model)
	params, err := store.ListHyperparameters(ctx, "model1", "marker", 2)
	assert.Equal(t, []string{}, params)
	assert.NoError(t, err)

	param := storage.Hyperparameters{
		ModelId:             "model1",
		HyperparametersId:   "param1",
		CanonicalCheckpoint: "canon1",
	}
	store.AddHyperparameters(ctx, param)

	param.HyperparametersId = "param3"
	store.AddHyperparameters(ctx, param)

	param.ModelId = "model2"
	param.HyperparametersId = "param1"
	store.AddHyperparameters(ctx, param)

	param.ModelId = "model1"
	param.HyperparametersId = "param2"
	store.AddHyperparameters(ctx, param)

	param.HyperparametersId = "param4"
	store.AddHyperparameters(ctx, param)

	param = storage.Hyperparameters{
		ModelId:             "model1",
		HyperparametersId:   "param1",
		CanonicalCheckpoint: "canon1",
	}

	params, err = store.ListHyperparameters(ctx, "model1", "marker", 2)
	assert.Equal(t, []string{"model1:param1", "model1:param2"}, params)
	assert.NoError(t, err)

	params, err = store.ListHyperparameters(ctx, "model1", "marker", 5)
	assert.Equal(t, []string{"model1:param1", "model1:param2", "model1:param3", "model1:param4"}, params)
	assert.NoError(t, err)

	params, err = store.ListHyperparameters(ctx, "model1", "param22", 5)
	assert.Equal(t, []string{"model1:param3", "model1:param4"}, params)
	assert.NoError(t, err)

	params, err = store.ListHyperparameters(ctx, "model2", "", 5)
	assert.Equal(t, []string{"model2:param1"}, params)
	assert.NoError(t, err)
}

func Test_UpdateHyperparams(t *testing.T, store storage.RepositoryStorage) {
	ctx := context.Background()

	model := storage.Model{
		ModelId:                  "model1",
		Details:                  "desc",
		CanonicalHyperparameters: "canon",
	}

	hyperparameters := storage.Hyperparameters{
		ModelId:             "model1",
		HyperparametersId:   "param1",
		CanonicalCheckpoint: "checkpoint1",
		Hyperparameters:     map[string]string{"hp1": "1"},
	}

	_, err := store.UpdateHyperparameters(ctx, hyperparameters)
	assert.Error(t, err)

	store.AddModel(ctx, model)
	_, err = store.UpdateHyperparameters(ctx, hyperparameters)
	assert.Error(t, err)

	store.AddHyperparameters(ctx, hyperparameters)

	hyperparametersUpdate := storage.Hyperparameters{
		ModelId:             "model1",
		HyperparametersId:   "param1",
		CanonicalCheckpoint: "",
		Hyperparameters:     nil,
	}

	updatedHyperparameters, err := store.UpdateHyperparameters(ctx, hyperparametersUpdate)
	assert.Equal(t, hyperparameters, updatedHyperparameters)

	hyperparametersUpdate.CanonicalCheckpoint = "checkpoint2"
	hyperparametersUpdate.Hyperparameters = make(map[string]string)
	hyperparametersUpdate.Hyperparameters["hp1"] = "1.1"
	hyperparametersUpdate.Hyperparameters["hp2"] = "2"
	hyperparametersUpdate.UpgradeTo = "upgradeTo1"
	expectedHyperparameters := storage.Hyperparameters{
		ModelId:             "model1",
		HyperparametersId:   "param1",
		CanonicalCheckpoint: "checkpoint2",
		UpgradeTo:           "upgradeTo1",
		Hyperparameters:     map[string]string{"hp1": "1.1", "hp2": "2"},
	}
	updatedHyperparameters, err = store.UpdateHyperparameters(ctx, hyperparametersUpdate)
	assert.Equal(t, expectedHyperparameters, updatedHyperparameters)
}

func Test_AddCheckpoint(t *testing.T, store storage.RepositoryStorage) {
	ctx := context.Background()

	checkpoint1 := storage.Checkpoint{
		ModelId:           "model1",
		HyperparametersId: "params1",
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
		Details:                  "desc",
		CanonicalHyperparameters: "canon",
	}
	store.AddModel(ctx, model1)

	_, err = store.GetCheckpoint(ctx, "model1", "params1", "cp1")
	assert.Error(t, err)

	err = store.AddCheckpoint(ctx, checkpoint1)
	assert.Error(t, err)

	params1 := storage.Hyperparameters{
		ModelId:             "model1",
		HyperparametersId:   "params1",
		CanonicalCheckpoint: "canon1",
		Hyperparameters:     map[string]string{"hp1": "1"},
	}
	store.AddHyperparameters(ctx, params1)

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
		Details:                  "desc",
		CanonicalHyperparameters: "canon",
	}
	store.AddModel(ctx, model1)

	_, err = store.ListCheckpoints(ctx, "model1", "params1", "", 4)
	assert.Error(t, err)

	params1 := storage.Hyperparameters{
		ModelId:             "model1",
		HyperparametersId:   "params1",
		CanonicalCheckpoint: "canon1",
		Hyperparameters:     map[string]string{"hp1": "1"},
	}
	store.AddHyperparameters(ctx, params1)
	_, err = store.ListCheckpoints(ctx, "model1", "params1", "", 4)
	assert.NoError(t, err)

	checkpoint1 := storage.Checkpoint{
		ModelId:           "model1",
		HyperparametersId: "params1",
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
