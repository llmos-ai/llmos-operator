package chat

import (
	"encoding/json"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/rancher/apiserver/pkg/store/empty"
	"github.com/rancher/apiserver/pkg/types"
	"github.com/sirupsen/logrus"
	"k8s.io/apiserver/pkg/authentication/user"

	"github.com/llmos-ai/llmos-controller/pkg/constant"
	entv1 "github.com/llmos-ai/llmos-controller/pkg/generated/ent"
	"github.com/llmos-ai/llmos-controller/pkg/utils"
)

type Store struct {
	empty.Store
	handler Handler
}

func toAPIObject(c *entv1.Chat) types.APIObject {
	return types.APIObject{
		Type:   "chat",
		ID:     c.ID.String(),
		Object: c,
	}
}

func (s *Store) ByID(apiOp *types.APIRequest, schema *types.APISchema, id string) (types.APIObject, error) {
	userInfo, ok := apiOp.GetUserInfo()
	if !ok {
		return types.APIObject{}, fmt.Errorf("failed to get user info")
	}

	chat, err := s.handler.FindByID(id, userInfo.GetUID())
	if err != nil {
		return types.APIObject{}, err
	}

	return types.APIObject{
		Type:   "chat",
		ID:     chat.ID.String(),
		Object: chat,
	}, nil
}

func (s *Store) List(apiOp *types.APIRequest, schema *types.APISchema) (types.APIObjectList, error) {
	userInfo, ok := apiOp.GetUserInfo()
	if !ok {
		return types.APIObjectList{}, fmt.Errorf("failed to get user info")
	}

	// listing all chats if is admin
	if isAdminOrSelf(userInfo, "") {
		return s.list("")
	}

	return s.list(userInfo.GetUID())
}

func (s *Store) list(uid string) (types.APIObjectList, error) {
	var err error
	var chats []*entv1.Chat
	if uid != "" {
		chats, err = s.handler.ListByUser(uid)
		if err != nil {
			return types.APIObjectList{}, err
		}
	} else {
		chats, err = s.handler.ListAll()
		if err != nil {
			return types.APIObjectList{}, err
		}
	}

	objs := make([]types.APIObject, 0)

	for _, chat := range chats {
		objs = append(objs, toAPIObject(chat))
	}

	return types.APIObjectList{
		Objects: objs,
	}, nil
}

func (s *Store) Create(apiOp *types.APIRequest, schema *types.APISchema, data types.APIObject) (types.APIObject, error) {
	userInfo, ok := apiOp.GetUserInfo()
	if !ok {
		return types.APIObject{}, fmt.Errorf("failed to get user info")
	}

	jsonData, err := json.Marshal(data.Object)
	if err != nil {
		return types.APIObject{}, fmt.Errorf("failed to encode data to json: %v", err)
	}

	// Decode the JSON into the struct
	var req NewChatRequest
	err = json.Unmarshal(jsonData, &req)
	if err != nil {
		return types.APIObject{}, fmt.Errorf("failed to decode data from json: %v", err)
	}
	logrus.Debugf("Creating new chat: %+v", req)

	chat, err := s.handler.Create(userInfo.GetUID(), req)
	if err != nil {
		return types.APIObject{}, fmt.Errorf("failed to create chat: %v", err)
	}
	return types.APIObject{
		Type:   "chat",
		ID:     chat.ID.String(),
		Object: chat,
	}, nil
}

func (s *Store) Update(apiOp *types.APIRequest, schema *types.APISchema, data types.APIObject, id string) (types.APIObject, error) {
	userInfo, ok := apiOp.GetUserInfo()
	if !ok {
		return types.APIObject{}, fmt.Errorf("failed to get user info")
	}

	jsonData, err := json.Marshal(data.Object)
	if err != nil {
		return types.APIObject{}, fmt.Errorf("failed to encode data to json: %v", err)
	}

	// Decode the JSON into the struct
	var req UpdateChatRequest
	err = json.Unmarshal(jsonData, &req)
	if err != nil {
		return types.APIObject{}, fmt.Errorf("failed to decode data from json: %v", err)
	}

	if !isAdminOrSelf(userInfo, req.UserId) {
		return types.APIObject{}, fmt.Errorf("unauthrozied to update the chat: %s", id)
	}

	logrus.Debugf("Updating existing chat %s: %+v", id, req)

	chat, err := s.handler.Update(id, req)
	if err != nil {
		return types.APIObject{}, fmt.Errorf("failed to create chat: %v", err)
	}
	return types.APIObject{
		Type:   "chat",
		ID:     chat.ID.String(),
		Object: chat,
	}, nil
}

func (s *Store) Delete(apiOp *types.APIRequest, schema *types.APISchema, id string) (types.APIObject, error) {
	err := s.handler.Delete(id)
	if err != nil {
		return types.APIObject{}, err
	}
	return types.APIObject{}, nil
}

func isAdminOrSelf(userInfo user.Info, uid string) bool {
	// check if user is admin and listing all chats
	if utils.ArrayStringContains(userInfo.GetGroups(), constant.AdminRole) {
		return true
	}

	if userInfo.GetUID() == uid {
		return true
	}
	return false
}
