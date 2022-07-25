import {combineReducers} from 'redux';

// import {Post} from 'mattermost-redux/types/posts';

function tokens(state: object = {}, action: {type: string, data: object}) {
    switch (action.type) {
    case "TOKEN_RECEIVED":
        let newTokens = {...state};
        console.log(state);
        // @ts-ignore
        newTokens[action.data.id] = action.data.jwt;
        console.log(newTokens);
        // @ts-ignore
        return action.data;
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
