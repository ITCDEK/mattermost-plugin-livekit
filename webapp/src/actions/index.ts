import {Store, Action, Dispatch} from 'redux';
import {useSelector} from 'react-redux';
import Client4 from 'mattermost-redux/client/client4';
import {DispatchFunc, GetStateFunc, ActionFunc, ActionResult} from 'mattermost-redux/types/actions';
import {getCurrentUserLocale} from 'mattermost-redux/selectors/entities/i18n';
import {getCurrentUser} from 'mattermost-redux/selectors/entities/users';

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
                dispatch({type: "TOKEN_RECEIVED", data: {id: postId, jwt: response}});
                dispatch({type: "GO_LIVE", data: postId});
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
                body: JSON.stringify({channel_id: channelId, message: getTranslation("room.topic")}),
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

export function getTranslation(id: string) {
    const currentUser = useSelector(getCurrentUser);
    const userName = currentUser.nickname;
    const locale = useSelector(getCurrentUserLocale);
    const templates = {
        "room.connect": {
            ru: "Войти",
            en: "Enter",
        },
        "room.topic": {
            ru: `${userName} создал(а) комнату для Вас`,
            en: `${userName} created live room for you`,
        },
    };
    console.log(currentUser);
    console.log(`locale = ${locale}`);
    // @ts-ignore
    return templates[id][locale];
}
