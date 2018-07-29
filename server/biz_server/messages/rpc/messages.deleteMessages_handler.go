/*
 *  Copyright (c) 2017, https://github.com/nebulaim
 *  All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package rpc

import (
	"github.com/golang/glog"
	"github.com/nebulaim/telegramd/baselib/grpc_util"
	"github.com/nebulaim/telegramd/baselib/logger"
	"github.com/nebulaim/telegramd/proto/mtproto"
	"github.com/nebulaim/telegramd/server/sync/sync_client"
	"golang.org/x/net/context"
)

// messages.deleteMessages#e58e95d2 flags:# revoke:flags.0?true id:Vector<int> = messages.AffectedMessages;
func (s *MessagesServiceImpl) MessagesDeleteMessages(ctx context.Context, request *mtproto.TLMessagesDeleteMessages) (*mtproto.Messages_AffectedMessages, error) {
	md := grpc_util.RpcMetadataFromIncoming(ctx)
	glog.Infof("messages.deleteMessages#e58e95d2 - metadata: %s, request: %s", logger.JsonDebugData(md), logger.JsonDebugData(request))

	var (
		deleteIdList = request.GetId()
	)

	deleteMessages := &mtproto.TLUpdateDeleteMessages{Data2: &mtproto.Update_Data{
		Messages: deleteIdList,
	}}

	state, err := sync_client.GetSyncClient().SyncOneUpdateData(md.AuthId, md.SessionId, md.UserId, deleteMessages.To_Update())
	if err != nil {
		return nil, err
	}

	affectedMessages := &mtproto.TLMessagesAffectedMessages{Data2: &mtproto.Messages_AffectedMessages_Data{
		Pts:      state.Pts,
		PtsCount: state.PtsCount,
	}}

	// 1. delete messages
	// 2. updateTopMessage
	if request.GetRevoke() {
		//  消息撤回
		deleteIdListMap := s.MessageModel.GetPeerDialogMessageIdList(md.UserId, request.GetId())
		glog.Info("messages.deleteMessages#e58e95d2 - deleteIdListMap: ", deleteIdListMap)
		for k, v := range deleteIdListMap {
			deleteMessages := &mtproto.TLUpdateDeleteMessages{Data2: &mtproto.Update_Data{
				Messages: v,
			}}
			sync_client.GetSyncClient().PushToUserOneUpdateData(k, deleteMessages.To_Update())
			s.MessageModel.DeleteByMessageIdList(k, v)
		}
		s.MessageModel.DeleteByMessageIdList(md.UserId, deleteIdList)
		// TODO(@benqi): 更新dialog的TopMessage
	} else {
		// 删除消息
		s.MessageModel.DeleteByMessageIdList(md.UserId, deleteIdList)

		// TODO(@benqi): 更新dialog的TopMessage
	}

	glog.Infof("messages.deleteMessages#e58e95d2 - reply: %s", logger.JsonDebugData(affectedMessages))
	return affectedMessages.To_Messages_AffectedMessages(), nil
}
