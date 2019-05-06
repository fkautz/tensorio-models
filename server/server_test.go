package server_test

import (
	"context"
	"fmt"
	"github.com/doc-ai/tensorio-models/api"
	"github.com/doc-ai/tensorio-models/server"
	"github.com/doc-ai/tensorio-models/storage/memory"
	"github.com/stretchr/testify/assert"
	"sort"
	"testing"
)

func testingServer() api.RepositoryServer {
	storage := memory.NewMemoryRepositoryStorage()
	srv := server.NewServer(storage)
	return srv
}

// Tests that models can be successfully created and listed against a fresh memory RepositoryStorage
// backend.
// Also tests that attempts to create duplicated models result in errors.
func TestCreateModelAndListModels(t *testing.T) {
	srv := testingServer()
	modelRequests := make([]api.CreateModelRequest, 5)
	for i := range modelRequests {
		modelID := fmt.Sprintf("test-model-%d", i)
		description := fmt.Sprintf("This is test model %d", i)
		model := api.CreateModelRequest{
			Model: &api.Model{
				ModelId:     modelID,
				Description: description,
			},
		}
		modelRequests[i] = model
	}

	ctx := context.Background()

	// TODO: Instead of doing ListModels on the server, we should inspect server.storage directly
	listModelsRequest := api.ListModelsRequest{
		MaxItems: 20,
	}
	for i, req := range modelRequests {
		listModelsResponse, err := srv.ListModels(ctx, &listModelsRequest)
		assert.NoError(t, err)
		models := listModelsResponse.ModelIds
		if len(models) != i {
			t.Errorf("Incorrect number of models in storage; expected: %d, actual: %d", i, len(models))
		}
		createModelResponse, err := srv.CreateModel(ctx, &req)
		if err != nil {
			t.Error(err)
		}
		expectedResourcePath := fmt.Sprintf("/models/%s", req.Model.ModelId)
		if createModelResponse.ResourcePath != expectedResourcePath {
			t.Errorf("Incorrect resource path for created model; expected: %s, actual: %s", expectedResourcePath, createModelResponse.ResourcePath)
		}
	}

	// Creation with a duplicated request should fail
	_, err := srv.CreateModel(ctx, &modelRequests[0])
	if err == nil {
		t.Error("Server did not error out on creation of duplicate model")
	}
}

// Tests that models are correctly listed (pagination behaviour)
func TestListModels(t *testing.T) {
	srv := testingServer()

	modelRequests := make([]api.CreateModelRequest, 21)
	for i := range modelRequests {
		modelID := fmt.Sprintf("test-model-%d", i)
		description := fmt.Sprintf("This is test model %d", i)
		model := api.CreateModelRequest{
			Model: &api.Model{
				ModelId:     modelID,
				Description: description,
			},
		}
		modelRequests[i] = model
	}
	ctx := context.Background()
	modelIDs := make([]string, len(modelRequests))
	for i, req := range modelRequests {
		modelIDs[i] = req.Model.ModelId
		_, err := srv.CreateModel(ctx, &req)
		if err != nil {
			t.Error(err)
		}
	}
	// NOTE: ModelIDs are sorted lexicographically, not chronologically!
	sort.Strings(modelIDs)

	type ListModelsTest struct {
		Server           *api.RepositoryServer
		Marker           string
		MaxItems         int32
		ExpectedModelIds []string
	}

	tests := []ListModelsTest{
		{
			Server:           &srv,
			MaxItems:         int32(5),
			ExpectedModelIds: modelIDs[0:5],
		},
		{
			Server:           &srv,
			Marker:           modelIDs[2],
			MaxItems:         int32(5),
			ExpectedModelIds: modelIDs[2:7],
		},
		{
			Server:           &srv,
			Marker:           modelIDs[16],
			MaxItems:         int32(5),
			ExpectedModelIds: modelIDs[16:21],
		},
		{
			Server:           &srv,
			Marker:           modelIDs[16],
			MaxItems:         int32(6),
			ExpectedModelIds: modelIDs[16:21],
		},
		// TODO(frederick): Specification says that list endpoints should return items AFTER marker,
		// not after and including marker. No need to change behaviour, just make the two consistent.
		{
			Server:           &srv,
			Marker:           modelIDs[0],
			MaxItems:         int32(20),
			ExpectedModelIds: modelIDs[0:20],
		},
	}

	for i, test := range tests {
		listModelsRequest := api.ListModelsRequest{
			Marker:   test.Marker,
			MaxItems: test.MaxItems,
		}

		tsrv := *test.Server
		listModelsResponse, err := tsrv.ListModels(ctx, &listModelsRequest)
		if err != nil {
			t.Error(err)
		}
		assert.Equalf(t, test.ExpectedModelIds, listModelsResponse.ModelIds, "TestListModels %d: ListModels request returned incorrect ModelIds", i)
	}
}

// Tests that model update behaviour is correct
func TestUpdateModel(t *testing.T) {
	srv := testingServer()

	model := api.CreateModelRequest{
		Model: &api.Model{
			ModelId:     "test-model",
			Description: "This is a test",
		},
	}

	ctx := context.Background()

	_, err := srv.CreateModel(ctx, &model)
	if err != nil {
		t.Error(err)
	}

	updateModelRequest := api.UpdateModelRequest{
		ModelId: model.Model.ModelId,
		Model: &api.Model{
			ModelId:     "test-model",
			Description: "This is only a test",
		},
	}
	updateModelResponse, err := srv.UpdateModel(ctx, &updateModelRequest)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, updateModelRequest.Model, updateModelResponse.Model, "UpdateModel models in request and response do not agree")
}

func TestMissingModelInHyperparameterUpdate(t *testing.T) {
	srv := testingServer()

	model := &api.CreateModelRequest{
		Model: &api.Model{
			ModelId:     "test-model",
			Description: "This is a test",
		},
	}

	srv.CreateModel(context.Background(), model)

	updateModelRequest := &api.UpdateModelRequest{
		ModelId: "test-model",
		Model:   nil,
	}

	updateModelResponse, err := srv.UpdateModel(context.Background(), updateModelRequest)
	assert.Nil(t, updateModelResponse)
	assert.Error(t, err)
}

// Creates a model and tests that GetModel returns the expected information
func TestGetModel(t *testing.T) {
	srv := testingServer()

	model := api.CreateModelRequest{
		Model: &api.Model{
			ModelId:     "test-model",
			Description: "This is a test",
		},
	}
	ctx := context.Background()
	_, err := srv.CreateModel(ctx, &model)
	if err != nil {
		t.Error(err)
	}

	getModelRequest := api.GetModelRequest{ModelId: model.Model.ModelId}
	getModelResponse, err := srv.GetModel(ctx, &getModelRequest)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, model.Model, getModelResponse.Model, "Did not receive the expected model in GetModel response")
}

// Tests that hyperparameters are correctly created
func TestCreateAndListHyperParameters(t *testing.T) {
	srv := testingServer()

	// Create a model under which to test hyperparameters functionality
	modelID := "test-model"
	model := api.CreateModelRequest{
		Model: &api.Model{
			ModelId:     modelID,
			Description: "This is a test",
		},
	}
	ctx := context.Background()
	_, err := srv.CreateModel(ctx, &model)
	if err != nil {
		t.Error(err)
	}

	createHyperparametersRequests := make([]api.CreateHyperParametersRequest, 21)
	for i := range createHyperparametersRequests {
		hyperparametersID := fmt.Sprintf("hyperparameters-%d", i)
		hyperparameters := make(map[string]string)
		hyperparameters["parameter"] = fmt.Sprintf("parameter-value-for-%d", i)
		createHyperparametersRequests[i] = api.CreateHyperParametersRequest{
			ModelId:           modelID,
			HyperParametersId: hyperparametersID,
			HyperParameters:   hyperparameters,
		}
	}

	listHyperparametersRequest := api.ListHyperParametersRequest{
		ModelId:  modelID,
		MaxItems: int32(21),
	}

	for i, req := range createHyperparametersRequests {
		listHyperparametersResponse, err := srv.ListHyperParameters(ctx, &listHyperparametersRequest)
		if err != nil {
			t.Error(err)
		}
		if len(listHyperparametersResponse.HyperParametersIds) != i {
			t.Errorf("Incorrect number of registered hyperparameters for model %s; expected: %d, actual: %d", modelID, i, len(listHyperparametersResponse.HyperParametersIds))
		}
		createHyperparametersResponse, err := srv.CreateHyperParameters(ctx, &req)
		if err != nil {
			t.Error(err)
		}
		expectedResourcePath := fmt.Sprintf("/models/%s/hyperparameters/%s", modelID, req.HyperParametersId)
		if createHyperparametersResponse.ResourcePath != expectedResourcePath {
			t.Errorf("Incorrect resource path in CreateHyperParameters response; expected: %s, actual: %s", expectedResourcePath, createHyperparametersResponse.ResourcePath)
		}
	}
}

// Tests that hyperparameters are correctly listed (pagination behaviour)
func TestListHyperParameters(t *testing.T) {
	srv := testingServer()

	// Create a model under which to test hyperparameters functionality
	modelID := "test-model"
	model := api.CreateModelRequest{
		Model: &api.Model{
			ModelId:     modelID,
			Description: "This is a test",
		},
	}
	ctx := context.Background()
	_, err := srv.CreateModel(ctx, &model)
	if err != nil {
		t.Error(err)
	}

	hpCreationRequests := make([]api.CreateHyperParametersRequest, 21)
	for i := range hpCreationRequests {
		hyperparametersID := fmt.Sprintf("hyperparameters-%d", i)
		hyperparameters := make(map[string]string)
		hyperparameters["parameter"] = fmt.Sprintf("parameter-value-for-%d", i)
		hpCreationRequests[i] = api.CreateHyperParametersRequest{
			ModelId:           modelID,
			HyperParametersId: hyperparametersID,
			HyperParameters:   hyperparameters,
		}
	}
	hyperparametersIDs := make([]string, len(hpCreationRequests))
	for i, req := range hpCreationRequests {
		hyperparametersIDs[i] = req.HyperParametersId
		_, err := srv.CreateHyperParameters(ctx, &req)
		if err != nil {
			t.Error(err)
		}
	}
	// NOTE: HyperParametersIDs are sorted lexicographically, not chronologically!
	sort.Strings(hyperparametersIDs)

	// ListHyperParameters does not return hyperparameters IDs, but rather tags of the form
	// <modelID>:<hyperparmetersID>
	// We account for this with hyperparametersTags
	hyperparametersTags := make([]string, len(hyperparametersIDs))
	for i, hyperparametersID := range hyperparametersIDs {
		hyperparametersTags[i] = fmt.Sprintf("%s:%s", modelID, hyperparametersID)
	}

	type ListHyperParametersTest struct {
		Server                     *api.RepositoryServer
		ModelId                    string
		Marker                     string
		MaxItems                   int32
		ExpectedHyperParametersIds []string
	}

	tests := []ListHyperParametersTest{
		{
			Server:                     &srv,
			ModelId:                    modelID,
			MaxItems:                   int32(5),
			ExpectedHyperParametersIds: hyperparametersTags[0:5],
		},
		{
			Server:                     &srv,
			ModelId:                    modelID,
			Marker:                     hyperparametersIDs[2],
			MaxItems:                   int32(5),
			ExpectedHyperParametersIds: hyperparametersTags[2:7],
		},
		{
			Server:                     &srv,
			ModelId:                    modelID,
			Marker:                     hyperparametersIDs[16],
			MaxItems:                   int32(5),
			ExpectedHyperParametersIds: hyperparametersTags[16:21],
		},
		{
			Server:                     &srv,
			ModelId:                    modelID,
			Marker:                     hyperparametersIDs[16],
			MaxItems:                   int32(6),
			ExpectedHyperParametersIds: hyperparametersTags[16:21],
		},
		// TODO(frederick): Specification says that list endpoints should return items AFTER marker,
		// not after and including marker. No need to change behaviour, just make the two consistent.
		{
			Server:                     &srv,
			ModelId:                    modelID,
			Marker:                     hyperparametersIDs[0],
			MaxItems:                   int32(20),
			ExpectedHyperParametersIds: hyperparametersTags[0:20],
		},
	}

	for i, test := range tests {
		listHyperParametersRequest := api.ListHyperParametersRequest{
			ModelId:  test.ModelId,
			Marker:   test.Marker,
			MaxItems: test.MaxItems,
		}

		tsrv := *test.Server
		listHyperParametersResponse, err := tsrv.ListHyperParameters(ctx, &listHyperParametersRequest)
		if err != nil {
			t.Error(err)
		}
		assert.Equalf(t, test.ExpectedHyperParametersIds, listHyperParametersResponse.HyperParametersIds, "TestListHyperParameters %d: ListHyperParameters request returned incorrect HyperParametersIds", i)
	}
}

// Tests that hyperparameters update behaviour is correct
func TestUpdateHyperParameters(t *testing.T) {
	srv := testingServer()

	// Create a model under which to test hyperparameters functionality
	modelID := "test-model"
	model := api.CreateModelRequest{
		Model: &api.Model{
			ModelId:     modelID,
			Description: "This is a test",
		},
	}
	ctx := context.Background()
	_, err := srv.CreateModel(ctx, &model)
	if err != nil {
		t.Error(err)
	}

	// Create hyperparameters to set up the test
	hyperparametersID := "test-hyperparameters"
	oldHyperparameters := make(map[string]string)
	oldHyperparameters["untouched-parameter-key"] = "old-value"
	oldHyperparameters["old-parameter-key"] = "old-value"

	hpCreationRequest := api.CreateHyperParametersRequest{
		ModelId:           modelID,
		HyperParametersId: hyperparametersID,
		HyperParameters:   oldHyperparameters,
	}
	_, err = srv.CreateHyperParameters(ctx, &hpCreationRequest)
	if err != nil {
		t.Error(err)
	}

	newHyperparameters := make(map[string]string)
	newHyperparameters["old-parameter-key"] = "new-value"
	newHyperparameters["new-parameter-key"] = "new-value"
	canonicalCheckpoint := "lol"
	hpUpdateRequest := api.UpdateHyperParametersRequest{
		ModelId:             modelID,
		HyperParametersId:   hyperparametersID,
		HyperParameters:     newHyperparameters,
		CanonicalCheckpoint: canonicalCheckpoint,
	}
	hpUpdateResponse, err := srv.UpdateHyperParameters(ctx, &hpUpdateRequest)
	if err != nil {
		t.Error(err)
	}

	// Note: UpdateHyperParameters merges hyperparameter maps from the request value into the
	// value in storage (with the former taking precedence on conflicting keys).
	expectedHyperparameters := make(map[string]string)
	for k, v := range oldHyperparameters {
		expectedHyperparameters[k] = v
	}
	for k, v := range newHyperparameters {
		expectedHyperparameters[k] = v
	}
	assert.Equal(t, modelID, hpUpdateResponse.ModelId, "Did not receive expected ModelID in UpdateHyperParameters response")
	assert.Equal(t, hyperparametersID, hpUpdateResponse.HyperParametersId, "Did not receive expected HyperParametersID in UpdateHyperParameters response")
	assert.Equal(t, canonicalCheckpoint, hpUpdateResponse.CanonicalCheckpoint, "Did not receive expected CanonicalCheckpoint in UpdateHyperParameters response")
	assert.Equal(t, expectedHyperparameters, hpUpdateResponse.HyperParameters, "Did not receive expected hyperparameters in UpdateHyperParameters response")
}

// Creates hyperparameters for a given model and tests that GetHyperParameters returns the expected information
func TestGetHyperParameters(t *testing.T) {
	srv := testingServer()

	// Create a model under which to test hyperparameters functionality
	modelID := "test-model"
	model := api.CreateModelRequest{
		Model: &api.Model{
			ModelId:     modelID,
			Description: "This is a test",
		},
	}
	ctx := context.Background()
	_, err := srv.CreateModel(ctx, &model)
	if err != nil {
		t.Error(err)
	}

	// Create hyperparameters to set up the test
	hyperparametersID := "test-hyperparameters"
	hyperparameters := make(map[string]string)
	hyperparameters["untouched-parameter-key"] = "old-value"
	hyperparameters["old-parameter-key"] = "old-value"

	hpCreationRequest := api.CreateHyperParametersRequest{
		ModelId:           modelID,
		HyperParametersId: hyperparametersID,
		HyperParameters:   hyperparameters,
	}
	_, err = srv.CreateHyperParameters(ctx, &hpCreationRequest)
	if err != nil {
		t.Error(err)
	}

	hpGetRequest := api.GetHyperParametersRequest{
		ModelId:           modelID,
		HyperParametersId: hyperparametersID,
	}
	hpGetResponse, err := srv.GetHyperParameters(ctx, &hpGetRequest)
	assert.NoError(t, err)
	assert.Equal(t, modelID, hpGetResponse.ModelId, "Did not receive expected ModelID in UpdateHyperParameters response")
	assert.Equal(t, hyperparametersID, hpGetResponse.HyperParametersId, "Did not receive expected HyperParametersID in UpdateHyperParameters response")
	assert.Equal(t, hyperparameters, hpGetResponse.HyperParameters, "Did not receive expected hyperparameters in UpdateHyperParameters response")
}

// Tests that checkpoints are correctly created and listed
func TestCreateAndListCheckpoints(t *testing.T) {
	srv := testingServer()

	// Create a model and hyperparameters under which to test checkpoint functionality
	modelID := "test-model"
	model := api.CreateModelRequest{
		Model: &api.Model{
			ModelId:     modelID,
			Description: "This is a test",
		},
	}
	ctx := context.Background()
	_, err := srv.CreateModel(ctx, &model)
	if err != nil {
		t.Error(err)
	}

	hyperparametersID := "test-hyperparameters"
	hyperparameters := make(map[string]string)
	hyperparameters["parameter"] = "parameter-value"

	hpCreationRequest := api.CreateHyperParametersRequest{
		ModelId:           modelID,
		HyperParametersId: hyperparametersID,
		HyperParameters:   hyperparameters,
	}
	hpCreationResponse, err := srv.CreateHyperParameters(ctx, &hpCreationRequest)
	if err != nil {
		t.Error(err)
	}
	hyperparametersResourcePath := hpCreationResponse.ResourcePath

	ckptCreationRequests := make([]api.CreateCheckpointRequest, 21)
	for i := range ckptCreationRequests {
		ckptID := fmt.Sprintf("checkpoint-%d", i)
		link := fmt.Sprintf("http://example.com/checkpoints-for-test/%d.zip", i)
		info := make(map[string]string)
		info["parameter"] = fmt.Sprintf("value-for-%d", i)
		ckptCreationRequests[i] = api.CreateCheckpointRequest{
			ModelId:           modelID,
			HyperParametersId: hyperparametersID,
			CheckpointId:      ckptID,
			Link:              link,
			Info:              info,
		}
	}

	listCheckpointsRequest := api.ListCheckpointsRequest{
		ModelId:           modelID,
		HyperParametersId: hyperparametersID,
		MaxItems:          int32(21),
	}

	for i, req := range ckptCreationRequests {
		listCheckpointsResponse, err := srv.ListCheckpoints(ctx, &listCheckpointsRequest)
		if err != nil {
			t.Error(err)
		}
		if len(listCheckpointsResponse.CheckpointIds) != i {
			t.Errorf("Incorrect number of registered hyperparameters for model %s; expected: %d, actual: %d", modelID, i, len(listCheckpointsResponse.CheckpointIds))
		}
		createCheckpointsResponse, err := srv.CreateCheckpoint(ctx, &req)
		if err != nil {
			t.Error(err)
		}
		expectedResourcePath := fmt.Sprintf("%s/checkpoints/%s", hyperparametersResourcePath, req.CheckpointId)
		if createCheckpointsResponse.ResourcePath != expectedResourcePath {
			t.Errorf("Incorrect resource path in CreateCheckpoints response; expected: %s, actual: %s", expectedResourcePath, createCheckpointsResponse.ResourcePath)
		}
	}
}

// Tests that checkpoints are correctly listed (pagination behaviour)
func TestListCheckpoints(t *testing.T) {
	srv := testingServer()

	// Create a model and hyperparameters under which to test checkpoint functionality
	modelID := "test-model"
	model := api.CreateModelRequest{
		Model: &api.Model{
			ModelId:     modelID,
			Description: "This is a test",
		},
	}
	ctx := context.Background()
	_, err := srv.CreateModel(ctx, &model)
	if err != nil {
		t.Error(err)
	}

	hyperparametersID := "test-hyperparameters"
	hyperparameters := make(map[string]string)
	hyperparameters["parameter"] = "parameter-value"

	hpCreationRequest := api.CreateHyperParametersRequest{
		ModelId:           modelID,
		HyperParametersId: hyperparametersID,
		HyperParameters:   hyperparameters,
	}
	_, err = srv.CreateHyperParameters(ctx, &hpCreationRequest)
	if err != nil {
		t.Error(err)
	}

	ckptCreationRequests := make([]api.CreateCheckpointRequest, 21)
	for i := range ckptCreationRequests {
		checkpointID := fmt.Sprintf("checkpoint-%d", i)
		link := fmt.Sprintf("http://example.com/checkpoints-for-test/%d.zip", i)
		info := make(map[string]string)
		info["parameter"] = fmt.Sprintf("value-for-%d", i)
		ckptCreationRequests[i] = api.CreateCheckpointRequest{
			ModelId:           modelID,
			HyperParametersId: hyperparametersID,
			CheckpointId:      checkpointID,
			Link:              link,
			Info:              info,
		}
	}

	checkpointIDs := make([]string, len(ckptCreationRequests))
	for i, req := range ckptCreationRequests {
		checkpointIDs[i] = req.CheckpointId
		_, err := srv.CreateCheckpoint(ctx, &req)
		if err != nil {
			t.Error(err)
		}
	}
	// NOTE: CheckpointIds are sorted lexicographically, not chronologically!
	sort.Strings(checkpointIDs)

	// ListCheckpoints does not return checkpoint IDs, but rather tags of the form
	// <modelID>:<hyperparmetersID>:<checkpointId>
	// We account for this with hyperparametersTags
	checkpointTags := make([]string, len(checkpointIDs))
	for i, checkpointID := range checkpointIDs {
		checkpointTags[i] = fmt.Sprintf("%s:%s:%s", modelID, hyperparametersID, checkpointID)
	}

	type ListCheckpointsTest struct {
		Server                *api.RepositoryServer
		ModelId               string
		HyperparametersId     string
		Marker                string
		MaxItems              int32
		ExpectedCheckpointIds []string
	}

	tests := []ListCheckpointsTest{
		{
			Server:                &srv,
			ModelId:               modelID,
			HyperparametersId:     hyperparametersID,
			MaxItems:              int32(5),
			ExpectedCheckpointIds: checkpointTags[0:5],
		},
		{
			Server:                &srv,
			ModelId:               modelID,
			HyperparametersId:     hyperparametersID,
			Marker:                checkpointIDs[2],
			MaxItems:              int32(5),
			ExpectedCheckpointIds: checkpointTags[2:7],
		},
		{
			Server:                &srv,
			ModelId:               modelID,
			HyperparametersId:     hyperparametersID,
			Marker:                checkpointIDs[16],
			MaxItems:              int32(5),
			ExpectedCheckpointIds: checkpointTags[16:21],
		},
		{
			Server:                &srv,
			ModelId:               modelID,
			HyperparametersId:     hyperparametersID,
			Marker:                checkpointIDs[16],
			MaxItems:              int32(6),
			ExpectedCheckpointIds: checkpointTags[16:21],
		},
		// TODO(frederick): Specification says that list endpoints should return items AFTER marker,
		// not after and including marker. No need to change behaviour, just make the two consistent.
		{
			Server:                &srv,
			ModelId:               modelID,
			HyperparametersId:     hyperparametersID,
			Marker:                checkpointIDs[0],
			MaxItems:              int32(20),
			ExpectedCheckpointIds: checkpointTags[0:20],
		},
	}

	for i, test := range tests {
		listCkptRequest := api.ListCheckpointsRequest{
			ModelId:           test.ModelId,
			HyperParametersId: test.HyperparametersId,
			Marker:            test.Marker,
			MaxItems:          test.MaxItems,
		}
		tsrv := *test.Server
		listCkptResponse, err := tsrv.ListCheckpoints(ctx, &listCkptRequest)
		if err != nil {
			t.Error(err)
		}
		errorMessage := fmt.Sprintf("Test %d: ListCheckpoints response does not contain the expected CheckpointIds", i)
		assert.Equalf(t, test.ExpectedCheckpointIds, listCkptResponse.CheckpointIds, errorMessage)
	}
}

// Creates a checkpoint for a given model and hyperparameters, and tests that GetCheckpoint returns the expected information
func TestGetCheckpoint(t *testing.T) {
	srv := testingServer()

	// Create a model and hyperparameters under which to test checkpoint functionality
	modelID := "test-model"
	model := api.CreateModelRequest{
		Model: &api.Model{
			ModelId:     modelID,
			Description: "This is a test",
		},
	}
	ctx := context.Background()
	_, err := srv.CreateModel(ctx, &model)
	if err != nil {
		t.Error(err)
	}

	hyperparametersID := "test-hyperparameters"
	hyperparameters := make(map[string]string)
	hyperparameters["parameter"] = "parameter-value"

	hpCreationRequest := api.CreateHyperParametersRequest{
		ModelId:           modelID,
		HyperParametersId: hyperparametersID,
		HyperParameters:   hyperparameters,
	}
	_, err = srv.CreateHyperParameters(ctx, &hpCreationRequest)
	if err != nil {
		t.Error(err)
	}

	checkpointID := "test-checkpoint"
	link := "http://example.com/checkpoints-for-test/ckpt.zip"
	info := make(map[string]string)
	info["parameter"] = "value"

	createCheckpointRequest := api.CreateCheckpointRequest{
		ModelId:           modelID,
		HyperParametersId: hyperparametersID,
		CheckpointId:      checkpointID,
		Link:              link,
		Info:              info,
	}
	createCheckpointResponse, err := srv.CreateCheckpoint(ctx, &createCheckpointRequest)
	if err != nil {
		t.Error(err)
	}

	getCheckpointRequest := api.GetCheckpointRequest{
		ModelId:           modelID,
		HyperParametersId: hyperparametersID,
		CheckpointId:      checkpointID,
	}
	getCheckpointResponse, err := srv.GetCheckpoint(ctx, &getCheckpointRequest)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, createCheckpointResponse.ResourcePath, getCheckpointResponse.ResourcePath, "Incorrect ResourcePath in GetCheckpointResponse")
	assert.Equal(t, link, getCheckpointResponse.Link, "Incorrect Link in GetCheckpointResponse")
	// TODO(frederick): Make the following assertion pass
	// assert.Equal(t, info, getCheckpointResponse.Info, "Incorrect Info in GetCheckpointResponse")
}
