import React from 'react';
import {Store, Action, Dispatch} from 'redux';

import {GlobalState} from 'mattermost-redux/types/store';
// import {DispatchFunc, GetStateFunc, ActionFunc, ActionResult} from 'mattermost-redux/types/actions';

import manifest from './manifest';
import reducer from './reducers';
import {postMeeting, getSettings, getTranslation} from './actions';
import ChannelHeaderIcon from './components/channel-header-icon';

// eslint-disable-next-line import/no-unresolved
import {PluginRegistry} from './types/mattermost-webapp';
import LivePost from './components/LivePost';

export default class LiveKitPlugin {
    // @ts-ignore
    store: Store;
    // eslint-disable-next-line @typescript-eslint/no-unused-vars, @typescript-eslint/no-empty-function
    public async initialize(registry: PluginRegistry, store: Store<GlobalState, Action<Record<string, unknown>>>) {
        console.log('-----------------------------------------------');
        console.log('was in initialize');
        console.log('-----------------------------------------------');

        // @ts-ignore
        window._registry = registry;

        // @ts-ignore
        window._store = store;
        this.store = store;
        registry.registerChannelHeaderButtonAction(
            <ChannelHeaderIcon/>,
            (channel) => {
                postMeeting(channel.id)(store.dispatch, store.getState);
            },
            getTranslation("icon.dropdown"),
            getTranslation("icon.tooltip"),
        );
        
        registry.registerSlashCommandWillBePostedHook(async (message, args) => {
            const fullCmd = message.trim();
            const Cmd = fullCmd.substring(0, fullCmd.indexOf(' '));
            let topicAndN = fullCmd.substring(fullCmd.indexOf(' ') + 1);
            if (Cmd !== '/liveroom') {
                return {message, args};
            }
            if (topicAndN[0] == '"') {
                topicAndN = topicAndN.substring(1);
                const quotesIndex = topicAndN.indexOf('"');
                if (quotesIndex > 0) {
                    // return {message: `/liveroom "${JSON.stringify(data)}"`, args};
                    return {message: fullCmd, args};
                }
            }
            return {error: {message: 'This command should be formatted as: /liveroom "room topic" N'}};
        });

        registry.registerPostTypeComponent('custom_livekit', LivePost);
        registry.registerReducer(reducer);
        store.dispatch(getSettings());

        // @see https://developers.mattermost.com/extend/plugins/webapp/reference/
    }
}

declare global {
    interface Window {
        registerPlugin(id: string, plugin: LiveKitPlugin): void
    }
}

window.registerPlugin(manifest.id, new LiveKitPlugin());
