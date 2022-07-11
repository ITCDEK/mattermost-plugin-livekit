import Client4 from 'mattermost-redux/client/client4';

import {id as pluginId} from '../manifest';
export function fetchToken(postId:string) {
    return async () => {
        try {
            console.log('fetchToken');
            console.log('postId', postId);
            const client = new Client4();

            client.doFetch(`/plugins/${pluginId}/mvp_token`, {
                body: JSON.stringify({post_id: postId}),
                method: 'POST',
                credentials: 'include',
            }).then((response) => {
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
