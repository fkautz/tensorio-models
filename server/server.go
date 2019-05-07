package server

import (
	"context"
	"fmt"
	"github.com/doc-ai/tensorio-models/api"
	"github.com/doc-ai/tensorio-models/storage"
	"github.com/doc-ai/tensorio-models/trace"
	"github.com/golang/protobuf/ptypes"
	"github.com/opentracing/opentracing-go"
	otlog "github.com/opentracing/opentracing-go/log"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strconv"
	"time"
)

type server struct {
	storage storage.RepositoryStorage
}

// NewServer - Creates an api.RepositoryServer which handles gRPC requests using a given
// storage.RepositoryStorage backend
func NewServer(storage storage.RepositoryStorage) api.RepositoryServer {
	return &server{storage: storage}
}

func (srv *server) Healthz(ctx context.Context, req *api.HealthCheckRequest) (*api.HealthCheckResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "server.Healthz")
	defer span.Finish()
	resp := &api.HealthCheckResponse{
		Status: 1,
	}
	return resp, nil
}

func (srv *server) ListModels(ctx context.Context, req *api.ListModelsRequest) (*api.ListModelsResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "server.ListModels")
	defer span.Finish()

	span.SetTag(trace.Marker, req.Marker)
	span.SetTag(trace.MaxItems, req.MaxItems)

	marker := req.Marker
	maxItems := int(req.MaxItems)
	if maxItems <= 0 {
		maxItems = 10
	}
	models, err := srv.storage.ListModels(ctx, marker, maxItems)
	if err != nil {
		span.LogFields(otlog.Error(err))
		span.SetTag("error", true)
		grpcErr := status.Error(codes.Unavailable, "Could not retrieve models from storage")
		return nil, grpcErr
	}
	res := &api.ListModelsResponse{
		ModelIds: models,
	}

	span.LogFields(otlog.String(trace.ItemsReturnedCount, strconv.Itoa(len(models))))

	return res, nil
}

func (srv *server) GetModel(ctx context.Context, req *api.GetModelRequest) (*api.GetModelResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "server.GetModel")
	defer span.Finish()

	span.SetTag(trace.ModelID, req.ModelId)

	modelID := req.ModelId
	model, err := srv.storage.GetModel(ctx, modelID)
	if err != nil {
		span.LogFields(otlog.Error(err))
		span.SetTag(trace.Error, true)

		message := fmt.Sprintf("Cloud not retrieve model (%s) from storage", modelID)
		grpcErr := status.Error(codes.Unavailable, message)

		return nil, grpcErr
	}
	respModel := api.Model{
		ModelId:                  model.ModelId,
		Details:                  model.Details,
		CanonicalHyperparameters: model.CanonicalHyperparameters,
	}
	resp := &api.GetModelResponse{
		Model: &respModel,
	}
	return resp, nil
}

func (srv *server) CreateModel(ctx context.Context, req *api.CreateModelRequest) (*api.CreateModelResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "server.CreateModel")
	defer span.Finish()

	span.SetTag(trace.ModelID, req.Model.ModelId)

	model := req.Model
	log.Printf("CreateModel request - Model: %v", model)
	storageModel := storage.Model{
		ModelId:                  model.ModelId,
		Details:                  model.Details,
		CanonicalHyperparameters: model.CanonicalHyperparameters,
	}
	err := srv.storage.AddModel(ctx, storageModel)
	if err != nil {
		span.LogFields(otlog.Error(err))
		span.SetTag(trace.Error, true)

		var grpcErr error
		if err == storage.ModelExistsError {
			grpcErr = status.Error(codes.AlreadyExists, fmt.Sprintf("model (%s) already exists", req.Model.ModelId))
		} else {
			grpcErr = status.Error(codes.Internal, "could not store model")
		}
		return nil, grpcErr
	}
	resourcePath := fmt.Sprintf("/models/%s", storageModel.ModelId)
	resp := &api.CreateModelResponse{ResourcePath: resourcePath}
	return resp, nil
}

func (srv *server) UpdateModel(ctx context.Context, req *api.UpdateModelRequest) (*api.UpdateModelResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "server.UpdateModel")
	defer span.Finish()

	span.SetTag(trace.ModelID, req.ModelId)

	modelID := req.ModelId
	model := req.Model
	if modelID == "" {
		err := api.MissingRequiredFieldError("modelId", "model id to update").Err()

		span.SetTag(trace.Error, true)
		span.LogFields(otlog.Error(err))

		return nil, err
	}
	if model == nil {
		err := api.MissingRequiredFieldError("model", "model to update").Err()

		span.SetTag(trace.Error, true)
		span.LogFields(otlog.Error(err))

		return nil, err
	}
	if req.Model.ModelId != "" {
		if req.ModelId != req.Model.ModelId {
			err := api.InvalidFieldValueError("request.model.modelId", "request.modelId != request.model.modelId").Err()

			span.SetTag(trace.Error, true)
			span.LogFields(otlog.Error(err))

			return nil, err
		}
	}
	if req.Model.ModelId != "" {
		if req.ModelId != req.Model.ModelId {
			return nil, api.InvalidFieldValueError("request.model.modelId", "request.modelId != request.model.modelId").Err()
		}
	}
	log.Printf("UpdateModel request - ModelId: %s, Model: %v", modelID, model)
	storedModel, err := srv.storage.GetModel(ctx, modelID)
	if err != nil {
		span.SetTag(trace.Error, true)
		span.LogFields(otlog.Error(err))

		message := fmt.Sprintf("Could not retrieve model (%s) from storage", modelID)
		grpcErr := status.Error(codes.Unavailable, message)

		return nil, grpcErr
	}
	updatedModel := storedModel
	if model.Details != "" {
		updatedModel.Details = model.Details
	}
	if model.CanonicalHyperparameters != "" {
		updatedModel.CanonicalHyperparameters = model.CanonicalHyperparameters
	}
	newlyStoredModel, err := srv.storage.UpdateModel(ctx, updatedModel)
	if err != nil {
		span.SetTag(trace.Error, true)
		span.LogFields(otlog.Error(err))

		message := fmt.Sprintf("Could not update model (%s) in storage", modelID)
		grpcErr := status.Error(codes.Unavailable, message)

		return nil, grpcErr
	}
	resp := &api.UpdateModelResponse{
		Model: &api.Model{
			ModelId:                  newlyStoredModel.ModelId,
			Details:                  newlyStoredModel.Details,
			CanonicalHyperparameters: newlyStoredModel.CanonicalHyperparameters,
		},
	}
	return resp, nil
}

func (srv *server) ListHyperparameters(ctx context.Context, req *api.ListHyperparametersRequest) (*api.ListHyperparametersResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "server.ListHyperparameters")
	defer span.Finish()

	span.SetTag(trace.ModelID, req.ModelId)
	span.SetTag(trace.Marker, req.Marker)
	span.SetTag(trace.MaxItems, req.MaxItems)

	modelID := req.ModelId
	marker := req.Marker
	maxItems := int(req.MaxItems)
	if maxItems <= 0 {
		maxItems = 10
	}
	log.Printf("ListHyperparameters request - ModelId: %s, Marker: %s, MaxItems: %d", modelID, marker, maxItems)
	hyperparametersIDs, err := srv.storage.ListHyperparameters(ctx, modelID, marker, maxItems)
	if err != nil {
		span.SetTag(trace.Error, true)
		span.LogFields(otlog.Error(err))

		message := fmt.Sprintf("Could not list hyperparameters for model (%s) in storage", modelID)
		grpcErr := status.Error(codes.Unavailable, message)

		return nil, grpcErr
	}
	resp := &api.ListHyperparametersResponse{
		ModelId:            modelID,
		HyperparametersIds: hyperparametersIDs,
	}
	return resp, nil
}

func (srv *server) CreateHyperparameters(ctx context.Context, req *api.CreateHyperparametersRequest) (*api.CreateHyperparametersResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "server.CreateHyperparameters")
	defer span.Finish()

	span.SetTag(trace.ModelID, req.ModelId)
	span.SetTag(trace.HyperparametersID, req.HyperparametersId)

	modelID := req.ModelId
	hyperparametersID := req.HyperparametersId
	canonicalCheckpoint := req.CanonicalCheckpoint
	hyperparameters := req.Hyperparameters
	storageHyperparameters := storage.Hyperparameters{
		ModelId:             modelID,
		HyperparametersId:   hyperparametersID,
		CanonicalCheckpoint: canonicalCheckpoint,
		Hyperparameters:     hyperparameters,
	}
	err := srv.storage.AddHyperparameters(ctx, storageHyperparameters)
	if err != nil {
		span.SetTag(trace.Error, true)
		span.LogFields(otlog.Error(err))

		message := fmt.Sprintf("Could not store hyperparameters (%v) in storage", storageHyperparameters)
		grpcErr := status.Error(codes.Unavailable, message)

		return nil, grpcErr
	}
	resourcePath := fmt.Sprintf("/models/%s/hyperparameters/%s", modelID, hyperparametersID)
	resp := &api.CreateHyperparametersResponse{
		ResourcePath: resourcePath,
	}
	return resp, nil
}

func (srv *server) GetHyperparameters(ctx context.Context, req *api.GetHyperparametersRequest) (*api.GetHyperparametersResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "server.GetHyperparameters")
	defer span.Finish()

	span.SetTag(trace.ModelID, req.ModelId)
	span.SetTag(trace.HyperparametersID, req.HyperparametersId)

	modelID := req.ModelId
	hyperparametersID := req.HyperparametersId
	storedHyperparameters, err := srv.storage.GetHyperparameters(ctx, modelID, hyperparametersID)
	if err != nil {
		span.SetTag(trace.Error, true)
		span.LogFields(otlog.Error(err))

		message := fmt.Sprintf("Could not get hyperparameters (%s) for model (%s) from storage", hyperparametersID, modelID)
		grpcErr := status.Error(codes.Unavailable, message)

		return nil, grpcErr
	}
	resp := &api.GetHyperparametersResponse{
		ModelId:             storedHyperparameters.ModelId,
		HyperparametersId:   storedHyperparameters.HyperparametersId,
		UpgradeTo:           storedHyperparameters.UpgradeTo,
		CanonicalCheckpoint: storedHyperparameters.CanonicalCheckpoint,
		Hyperparameters:     storedHyperparameters.Hyperparameters,
	}
	return resp, nil
}

func (srv *server) UpdateHyperparameters(ctx context.Context, req *api.UpdateHyperparametersRequest) (*api.UpdateHyperparametersResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "server.UpdateHyperparameters")
	defer span.Finish()

	span.SetTag(trace.ModelID, req.ModelId)
	span.SetTag(trace.HyperparametersID, req.HyperparametersId)

	modelID := req.ModelId
	hyperparametersID := req.HyperparametersId
	upgradeTo := req.UpgradeTo
	canonicalCheckpoint := req.CanonicalCheckpoint
	hyperparameters := req.Hyperparameters

	existingHyperparameters, err := srv.storage.GetHyperparameters(ctx, modelID, hyperparametersID)
	if err != nil {
		span.SetTag(trace.Error, true)
		span.LogFields(otlog.Error(err))

		message := fmt.Sprintf("Could not get hyperparameters (%s) for model (%s) from storage", hyperparametersID, modelID)
		grpcErr := status.Error(codes.Unavailable, message)

		return nil, grpcErr
	}

	updatedHyperparameters := existingHyperparameters
	if canonicalCheckpoint != "" {
		updatedHyperparameters.CanonicalCheckpoint = canonicalCheckpoint
	}
	if upgradeTo != "" {
		updatedHyperparameters.UpgradeTo = upgradeTo
	}
	for k, v := range hyperparameters {
		updatedHyperparameters.Hyperparameters[k] = v
	}
	storedHyperparameters, err := srv.storage.UpdateHyperparameters(ctx, updatedHyperparameters)
	if err != nil {
		span.SetTag(trace.Error, true)
		span.LogFields(otlog.Error(err))
		message := fmt.Sprintf("Could not store hyperparameters (%v) in storage", updatedHyperparameters)
		grpcErr := status.Error(codes.Unavailable, message)
		return nil, grpcErr
	}

	resp := &api.UpdateHyperparametersResponse{
		ModelId:             storedHyperparameters.ModelId,
		HyperparametersId:   storedHyperparameters.HyperparametersId,
		UpgradeTo:           "",
		CanonicalCheckpoint: storedHyperparameters.CanonicalCheckpoint,
		Hyperparameters:     storedHyperparameters.Hyperparameters,
	}
	return resp, nil
}

func (srv *server) ListCheckpoints(ctx context.Context, req *api.ListCheckpointsRequest) (*api.ListCheckpointsResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "server.ListCheckpoints")
	defer span.Finish()

	span.SetTag(trace.ModelID, req.ModelId)
	span.SetTag(trace.HyperparametersID, req.HyperparametersId)
	span.SetTag(trace.Marker, req.Marker)
	span.SetTag(trace.MaxItems, req.MaxItems)

	modelID := req.ModelId
	hyperparametersID := req.HyperparametersId
	marker := req.Marker
	maxItems := int(req.MaxItems)
	if maxItems <= 0 {
		maxItems = 10
	}
	checkpointIDs, err := srv.storage.ListCheckpoints(ctx, modelID, hyperparametersID, marker, maxItems)
	if err != nil {
		span.SetTag(trace.Error, true)
		span.LogFields(otlog.Error(err))

		message := fmt.Sprintf("Could not list checkpoints for model (%s) and hyperparameters (%s) in storage", modelID, hyperparametersID)
		grpcErr := status.Error(codes.Unavailable, message)

		return nil, grpcErr
	}
	resp := &api.ListCheckpointsResponse{
		CheckpointIds: checkpointIDs,
	}
	return resp, nil
}

func (srv *server) CreateCheckpoint(ctx context.Context, req *api.CreateCheckpointRequest) (*api.CreateCheckpointResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "server.CreateCheckpoint")
	defer span.Finish()

	span.SetTag(trace.ModelID, req.ModelId)
	span.SetTag(trace.HyperparametersID, req.HyperparametersId)
	span.SetTag(trace.Marker, req.CheckpointId)

	modelID := req.ModelId
	hyperparametersID := req.HyperparametersId
	checkpointID := req.CheckpointId
	link := req.Link

	storageCheckpoint := storage.Checkpoint{
		ModelId:           modelID,
		HyperparametersId: hyperparametersID,
		CheckpointId:      checkpointID,
		CreatedAt:         time.Now(),
		Link:              link,
		Info:              req.Info,
	}

	err := srv.storage.AddCheckpoint(ctx, storageCheckpoint)
	if err != nil {
		span.SetTag(trace.Error, true)
		span.LogFields(otlog.Error(err))

		message := fmt.Sprintf("Could not store checkpoint (%v) in storage", storageCheckpoint)
		grpcErr := status.Error(codes.Unavailable, message)

		return nil, grpcErr
	}

	resourcePath := getCheckpointResourcePath(modelID, hyperparametersID, checkpointID)
	resp := &api.CreateCheckpointResponse{
		ResourcePath: resourcePath,
	}

	return resp, nil
}

func (srv *server) GetCheckpoint(ctx context.Context, req *api.GetCheckpointRequest) (*api.GetCheckpointResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "server.GetCheckpoint")
	defer span.Finish()

	span.SetTag(trace.ModelID, req.ModelId)
	span.SetTag(trace.HyperparametersID, req.HyperparametersId)
	span.SetTag(trace.Marker, req.CheckpointId)

	modelID := req.ModelId
	hyperparametersID := req.HyperparametersId
	checkpointID := req.CheckpointId
	log.Printf("GetCheckpoint request - ModelId: %s, HyperparametersId: %s, CheckpointId: %s", modelID, hyperparametersID, checkpointID)
	storedCheckpoint, err := srv.storage.GetCheckpoint(ctx, modelID, hyperparametersID, checkpointID)
	if err != nil {
		span.SetTag(trace.Error, true)
		span.LogFields(otlog.Error(err))

		message := fmt.Sprintf("Could not get checkpoint (%s) of hyperparameters (%s) for model (%s) from storage", checkpointID, hyperparametersID, modelID)
		grpcErr := status.Error(codes.Unavailable, message)

		return nil, grpcErr
	}
	resourcePath := getCheckpointResourcePath(modelID, hyperparametersID, checkpointID)
	createdAt, err := ptypes.TimestampProto(storedCheckpoint.CreatedAt)
	if err != nil {
		span.SetTag(trace.Error, true)
		span.LogFields(otlog.Error(err))

		message := "unable to serialize createdAt timestamp"
		grpcErr := status.Error(codes.Internal, message)

		return nil, grpcErr
	}

	resp := &api.GetCheckpointResponse{
		ResourcePath: resourcePath,
		Link:         storedCheckpoint.Link,
		CreatedAt:    createdAt,
		Info:         storedCheckpoint.Info,
	}

	return resp, nil
}

func getCheckpointResourcePath(modelID, hyperparametersID, checkpointID string) string {
	resourcePath := fmt.Sprintf("/models/%s/hyperparameters/%s/checkpoints/%s", modelID, hyperparametersID, checkpointID)
	return resourcePath
}
