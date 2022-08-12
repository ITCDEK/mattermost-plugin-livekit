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

            client.doFetch(`/plugins/${pluginId}/join`, {
                body: JSON.stringify({post_id: postId}),
                method: 'POST',
                credentials: 'include',
            }).then((response) => {
                // @ts-ignore
                if (response.status == "OK") {
                    // @ts-ignore
                    dispatch({type: "TOKEN_RECEIVED", data: {id: postId, jwt: response.data}});
                    dispatch({type: "GO_LIVE", data: postId});
                } else {
                    // @ts-ignore
                    console.log(`Token error: ${response.error}`);
                }
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
    return async (dispatch: DispatchFunc, getState: GetStateFunc): Promise<ActionResult> => {
        try {
            const client = new Client4();
            client.doFetch(`/plugins/${pluginId}/create`, {
                body: JSON.stringify({channel_id: channelId, message: getTranslation("room.topic")}),
                method: 'POST',
                credentials: 'include',
            }).then((response) => {
                // @ts-ignore
                if (response.status == "OK") {
                    dispatch({type: "ROOM_HOSTED", data: response});
                } else {
                    // @ts-ignore
                    console.log(`Hosting room error: ${response.error}`);
                }
            });
            return {data: "Ok"};
        } catch (error) {
            return {error};
        }
    };
}

export function deletePost(postId:string): ActionFunc {
    return async (dispatch: DispatchFunc, getState: GetStateFunc): Promise<ActionResult> => {
        try {
            const client = new Client4();
            client.doFetch(`/plugins/${pluginId}/delete`, {
                body: JSON.stringify({post_id: postId}),
                method: 'POST',
                credentials: 'include',
            }).then((response) => {
                // @ts-ignore
                console.log('Post deleting response:', response);
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
    // @ts-ignore
    const state = window.plugins[pluginId].store.getState();
    const currentUser = getCurrentUser(state);
    let userName = currentUser.nickname;
    if (userName == "") userName = `${currentUser.first_name} ${currentUser.last_name}`;
    // console.log(currentUser);
    const supportedLocales = ["en", "ru"];
    let locale = getCurrentUserLocale(state);
    // console.log(`locale = ${locale}`);
    locale = supportedLocales.includes(locale) ? locale : "en";
    const templates = {
        "icon.dropdown": {
            ru: "Создать видео пост",
            en: 'Start LiveKit Meeting',
        },
        "icon.tooltip": {
            ru: "Создать встречу вживую",
            en: 'Start LiveKit Meeting',
        },
        "room.connect": {
            ru: "Войти",
            en: "Enter",
        },
        "room.topic": {
            ru: `${userName} приглашает в свою комнату`,
            en: `${userName} created live room`,
        },
        "status.connecting": {
            ru: "Подключаемся...",
            en: "Connecting...",
        },
        "status.noRoom": {
            ru: "Комната закрыта",
            en: "Room is closed",
        }
    };
    // @ts-ignore
    return templates[id][locale];
}
