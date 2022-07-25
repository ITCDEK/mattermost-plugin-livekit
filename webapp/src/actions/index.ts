import {Store, Action, Dispatch} from 'redux';
import Client4 from 'mattermost-redux/client/client4';
import {DispatchFunc, GetStateFunc, ActionFunc, ActionResult} from 'mattermost-redux/types/actions';

import {id as pluginId} from '../manifest';
export function fetchToken(postId:string) {
    return async () => {
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
                console.log('_______________response______________');
                console.log(response);
                console.log('_______________response______________');
            });
        } catch (error) {
            // eslint-disable-next-line no-alert
            alert(`Ошибка ${error}`);
            // eslint-disable-next-line no-console
            console.error(error);
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
                body: JSON.stringify({}),
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