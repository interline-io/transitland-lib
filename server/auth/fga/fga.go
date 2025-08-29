package fga

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"strings"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/server/auth/authz"
	openfga "github.com/openfga/go-sdk"
)

func FromFGATupleKey(fgatk openfga.TupleKey) authz.TupleKey {
	rel, _ := authz.RelationString(*fgatk.Relation)
	act, _ := authz.ActionString(*fgatk.Relation)
	return authz.TupleKey{
		Subject:  NewEntityKeySplit(*fgatk.User),
		Object:   NewEntityKeySplit(*fgatk.Object),
		Relation: rel,
		Action:   act,
	}
}

func ToFGATupleKey(tk authz.TupleKey) openfga.TupleKey {
	fgatk := openfga.TupleKey{}
	if tk.Subject.Name != "" {
		fgatk.User = openfga.PtrString(tk.Subject.String())
	} else if authz.IsObjectType(tk.Subject.Type) {
		fgatk.Object = openfga.PtrString(tk.Subject.Type.String() + ":")
	}
	if tk.Object.Name != "" {
		fgatk.Object = openfga.PtrString(tk.Object.String())
	} else if authz.IsObjectType(tk.Object.Type) {
		fgatk.Object = openfga.PtrString(tk.Object.Type.String() + ":")
	}
	if authz.IsAction(tk.Action) {
		fgatk.Relation = openfga.PtrString(tk.Action.String())
	} else if authz.IsRelation(tk.Relation) {
		fgatk.Relation = openfga.PtrString(tk.Relation.String())
	}
	return fgatk
}

func NewEntityKeySplit(v string) authz.EntityKey {
	ret := authz.EntityKey{}
	a := strings.Split(v, ":")
	if len(a) > 1 {
		ret.Type, _ = authz.ObjectTypeString(a[0])
		ret.Name = a[1]
	} else if len(a) > 0 {
		ret.Type, _ = authz.ObjectTypeString(a[0])
	}
	ns := strings.Split(ret.Name, "#")
	if len(ns) > 1 {
		ret.Name = ns[0]
		ret.RefRel, _ = authz.RelationString(ns[1])
	}
	return ret
}

// //////////

type FGAClient struct {
	ModelID string
	client  *openfga.APIClient
}

func NewFGAClient(endpoint string, storeId string, modelId string) (*FGAClient, error) {
	ep, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	cfg, err := openfga.NewConfiguration(openfga.Configuration{
		ApiScheme: ep.Scheme,
		ApiHost:   ep.Host,
		StoreId:   storeId,
	})
	if err != nil {
		return nil, err
	}
	apiClient := openfga.NewAPIClient(cfg)
	return &FGAClient{
		ModelID: modelId,
		client:  apiClient,
	}, nil
}

func (c *FGAClient) Check(ctx context.Context, tk authz.TupleKey, ctxTuples ...authz.TupleKey) (bool, error) {
	if err := tk.Validate(); err != nil {
		return false, err
	}
	var fgaCtxTuples openfga.ContextualTupleKeys
	for _, ctxTuple := range ctxTuples {
		fgaCtxTuples.TupleKeys = append(fgaCtxTuples.TupleKeys, ToFGATupleKey(ctxTuple))
	}
	body := openfga.CheckRequest{
		AuthorizationModelId: openfga.PtrString(c.ModelID),
		TupleKey:             ToFGATupleKey(tk),
		ContextualTuples:     &fgaCtxTuples,
	}
	data, _, err := c.client.OpenFgaApi.Check(context.Background()).Body(body).Execute()
	if err != nil {
		return false, err
	}
	return data.GetAllowed(), nil
}

func (c *FGAClient) ListObjects(ctx context.Context, tk authz.TupleKey) ([]authz.TupleKey, error) {
	body := openfga.ListObjectsRequest{
		AuthorizationModelId: openfga.PtrString(c.ModelID),
		User:                 tk.Subject.String(),
		Relation:             tk.ActionOrRelation(),
		Type:                 tk.Object.Type.String(),
	}
	data, _, err := c.client.OpenFgaApi.ListObjects(context.Background()).Body(body).Execute()
	if err != nil {
		return nil, err
	}
	var ret []authz.TupleKey
	for _, v := range data.GetObjects() {
		ret = append(ret, authz.TupleKey{
			Subject: authz.NewEntityKey(tk.Subject.Type, tk.Subject.Name),
			Object:  NewEntityKeySplit(v),
			Action:  tk.Action,
		})
	}
	return ret, nil
}

func (c *FGAClient) GetObjectTuples(ctx context.Context, tk authz.TupleKey) ([]authz.TupleKey, error) {
	if err := tk.Validate(); err != nil {
		return nil, err
	}
	fgatk := ToFGATupleKey(tk)
	body := openfga.ReadRequest{
		TupleKey: &fgatk,
	}
	data, _, err := c.client.OpenFgaApi.Read(ctx).Body(body).Execute()
	if err != nil {
		return nil, err
	}
	var ret []authz.TupleKey
	for _, t := range *data.Tuples {
		ret = append(ret, FromFGATupleKey(t.GetKey()))
	}
	return ret, nil
}

func (c *FGAClient) SetExclusiveRelation(ctx context.Context, tk authz.TupleKey) error {
	return c.replaceTuple(ctx, tk, false, tk.Relation)

}

func (c *FGAClient) SetExclusiveSubjectRelation(ctx context.Context, tk authz.TupleKey, checkRelations ...authz.Relation) error {
	return c.replaceTuple(ctx, tk, true, checkRelations...)
}

func (c *FGAClient) replaceTuple(ctx context.Context, tk authz.TupleKey, checkSubjectEqual bool, checkRelations ...authz.Relation) error {
	if err := tk.Validate(); err != nil {
		log.Error().Err(err).Str("tk", tk.String()).Msg("replaceTuple")
		return err
	}
	relTypeOk := false
	for _, checkRel := range checkRelations {
		if tk.Relation == checkRel {
			relTypeOk = true
		}
	}
	if !relTypeOk {
		return fmt.Errorf("unknown relation %s for types %s and %s", tk.Relation.String(), tk.Subject.Type.String(), tk.Object.Type.String())
	}
	log.Trace().Str("tk", tk.String()).Msg("replaceTuple")

	currentTks, err := c.GetObjectTuples(ctx, authz.NewTupleKey().WithObject(tk.Object.Type, tk.Object.Name))
	if err != nil {
		return err
	}

	var matchTks []authz.TupleKey
	var delTks []authz.TupleKey
	for _, checkTk := range currentTks {
		relMatch := false
		for _, r := range checkRelations {
			if checkTk.Relation == r {
				relMatch = true
			}
		}
		if !relMatch {
			continue
		}
		if checkSubjectEqual && !checkTk.Subject.Equals(tk.Subject) {
			continue
		}
		if checkTk.Equals(tk) {
			matchTks = append(matchTks, checkTk)
		} else {
			delTks = append(delTks, checkTk)
		}
	}

	// Write new tuple before deleting others
	if len(matchTks) == 0 {
		if err := c.WriteTuple(ctx, tk); err != nil {
			return err
		}
	}

	// Delete exsiting tuples
	var errs []error
	for _, delTk := range delTks {
		if err := c.DeleteTuple(ctx, delTk); err != nil {
			errs = append(errs, err)
		}
	}
	for _, err := range errs {
		log.Trace().Err(err).Str("tk", tk.String()).Msg("replaceTuple")
	}
	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

func (c *FGAClient) WriteTuple(ctx context.Context, tk authz.TupleKey) error {
	if err := tk.Validate(); err != nil {
		log.Error().Str("tk", tk.String()).Msg("WriteTuple")
		return err
	}
	log.Trace().Str("tk", tk.String()).Msg("WriteTuple")
	body := openfga.WriteRequest{
		Writes:               &openfga.TupleKeys{TupleKeys: []openfga.TupleKey{ToFGATupleKey(tk)}},
		AuthorizationModelId: openfga.PtrString(c.ModelID),
	}
	_, _, err := c.client.OpenFgaApi.Write(context.Background()).Body(body).Execute()
	return err
}

func (c *FGAClient) DeleteTuple(ctx context.Context, tk authz.TupleKey) error {
	if err := tk.Validate(); err != nil {
		log.Error().Err(err).Str("tk", tk.String()).Msg("DeleteTuple")
		return err
	}
	log.Trace().Str("tk", tk.String()).Msg("DeleteTuple")
	body := openfga.WriteRequest{
		Deletes:              &openfga.TupleKeys{TupleKeys: []openfga.TupleKey{ToFGATupleKey(tk)}},
		AuthorizationModelId: openfga.PtrString(c.ModelID),
	}
	_, _, err := c.client.OpenFgaApi.Write(context.Background()).Body(body).Execute()
	return err
}

func (c *FGAClient) CreateStore(ctx context.Context, storeName string) (string, error) {
	// Create new store
	resp, _, err := c.client.OpenFgaApi.CreateStore(context.Background()).Body(openfga.CreateStoreRequest{
		Name: storeName,
	}).Execute()
	if err != nil {
		return "", err
	}
	storeId := resp.GetId()
	log.Info().Msgf("created store: %s", storeId)
	c.client.SetStoreId(storeId)
	return storeId, nil
}

func (c *FGAClient) CreateModel(ctx context.Context, fn string) (string, error) {
	// Create new model
	var dslJson []byte
	var err error
	if dslJson, err = ioutil.ReadFile(fn); err != nil {
		return "", err
	}
	var body openfga.WriteAuthorizationModelRequest
	if err := json.Unmarshal(dslJson, &body); err != nil {
		return "", err
	}
	modelId := ""
	if resp, _, err := c.client.OpenFgaApi.WriteAuthorizationModel(context.Background()).Body(body).Execute(); err != nil {
		return "", err
	} else {
		modelId = resp.GetAuthorizationModelId()
	}
	log.Info().Msgf("created model: %s", modelId)
	return modelId, nil
}
