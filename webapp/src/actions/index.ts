import {Store, Action, Dispatch} from 'redux';
import Client4 from 'mattermost-redux/client/client4';
import {DispatchFunc, GetStateFunc, ActionFunc, ActionResult} from 'mattermost-redux/types/actions';

import {id as pluginId} from '../manifest';
export function fetchToken(postId:string): ActionFunc {
    return async (dispatch: DispatchFunc): Promise<ActionResult> => {
        try {
            console.log('fetchToken call with postId =', postId);
            const client = new Client4();

            client.doFetch(`/plugins/${pluginId}/mvp_token`, {
                body: JSON.stringify({post_id: postId}),
                method: 'POST',
                credentials: 'include',
            }).then((response) => {
                // @ts-ignore
                window.__token = response;
                console.log('______token response______');
                console.log(response);
                console.log('______end of response_____');
                dispatch({type: "TOKEN_RECEIVED", data: {id: postId, jwt: response}});
            });
            return {data: "Ok"};
        } catch (error) {
            // eslint-disable-next-line no-alert
            alert(`Ошибка ${error}`);
            // eslint-disable-next-line no-console
            console.error(error);
            return {error};
        }
    };
}

export function postMeeting(channelId:string): ActionFunc {
    return async (dispatch: DispatchFunc): Promise<ActionResult> => {
        try {
            const client = new Client4();
            client.doFetch(`/plugins/${pluginId}/room`, {
                body: JSON.stringify({channel_id: channelId}),
                method: 'POST',
                credentials: 'include',
            }).then((response) => {
                // @ts-ignore
                console.log('Hosting room:');
                console.log(response);
                dispatch({
                    type: "ROOM_HOSTED",
                    data: response
                });
            });
            return {data: "Ok"};
        } catch (error) {
            return {error};
        }
    };
}

export function getSettings(): ActionFunc {
    return async (dispatch: DispatchFunc): Promise<ActionResult> => {
        try {
            const client = new Client4();
            client.doFetch(`/plugins/${pluginId}/settings`, {
                method: 'GET',
                credentials: 'include',
            }).then((response) => {
                // @ts-ignore
                console.log('Got these settings:');
                console.log(response);
                dispatch({
                    type: "CONFIG_RECEIVED",
                    data: response
                });
            });
            return {data: "Ok"};
        } catch (error) {
            return {error};
        }
    };
}