import {combineReducers} from 'redux';

// import {Post} from 'mattermost-redux/types/posts';

function tokens(state: object = {}, action: {type: string, data: object}) {
    switch (action.type) {
    case "STORE":
        let newTokens = {...state};
        for (let post_id of Object.keys(action.data)) {
            // @ts-ignore
            newTokens[post_id] = action.data[post_id]
        }
        return newTokens;
    default:
        return state;
    }
}

function config(state: object = {}, action: {type: string, data: object}) {
    switch (action.type) {
    case "CONFIG_RECEIVED":
        console.log('config reducer in action!');
        return action.data;
    default:
        return state;
    }
}

export default combineReducers({
    tokens,
    config
});
