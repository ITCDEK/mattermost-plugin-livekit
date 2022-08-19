import {combineReducers} from 'redux';

// import {Post} from 'mattermost-redux/types/posts';

function liveRooms(state: object = {}, action: {type: string, data: string}) {
    let newLiveSet = {...state};
    switch (action.type) {
    case "GO_LIVE":
        // @ts-ignore
        newLiveSet[action.data] = true;
        console.log("liveRooms:", newLiveSet);
        return newLiveSet;
    case "GO_STILL":
        // @ts-ignore
        newLiveSet[action.data] = false;
        console.log("liveRooms:", newLiveSet);
        return newLiveSet;
    default:
        return state;
    }
}

function tokens(state: object = {}, action: {type: string, data: object}) {
    switch (action.type) {
    case "TOKEN_RECEIVED":
        let newSet = {...state};
        // @ts-ignore
        newSet[action.data.id] = action.data.jwt;
        console.log(newSet);
        return newSet;
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
    liveRooms,
    tokens,
    config
});
