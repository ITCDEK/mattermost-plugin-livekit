import React from 'react';
import {connect, useSelector, useDispatch} from 'react-redux';
import {getTheme} from 'mattermost-redux/selectors/entities/preferences';
import {makeStyleFromTheme} from 'mattermost-redux/utils/theme_utils';

import {fetchToken, getTranslation} from '../actions';
import {id as pluginId} from '../manifest';

const StillRoom = (props) => {
    const dispatch = useDispatch();
    const buttonLabel = getTranslation("room.connect");
    const style = getStyle(props.theme);
    const goLive = () => props.token ? dispatch({type: "GO_LIVE", data: props.post.id}) : dispatch(fetchToken(props.post.id));
    return (
        <div style={style.wrapper} onClick = {props.stopPropagation}>
            <div style={style.message}>{props.post.message}</div>
            <div style={style.buttonWrapper}>
                <div style={style.connectButton} className = "btn btn-lg btn-primary" onClick = {goLive}>{buttonLabel}</div>
            </div>
        </div>
    );
}

const getStyle = makeStyleFromTheme((theme) => {
    console.log(theme);
    var styles = {
        wrapper: {
            width: "100%",
            display: "flex",
            marginTop: "2vh",
        },
        message: {
            width: "80%",
            borderLeftStyle: 'solid',
            borderLeftWidth: '4px',
            padding: '10px',
            borderLeftColor: '#89AECB'
        },
        buttonWrapper: {
            width: "20%",
            display: "flex",
            justifyContent: "center",
        },
        connectButton: {
            color: theme.buttonColor,
            // backgroundColor: theme.buttonBg,
            backgroundColor: "coral",
        },
    };
    //Denim, Sapphire, Quartz, Indigo, Onyx
    if (theme.type == "Indigo" || theme.type == "Onyx") {}
    return styles;
});

function mapStateToProps(state, ownProps) {
    return {
        ...ownProps,
        // theme: getTheme(state),
        tokens: state[`plugins-${pluginId}`].tokens,
        pluginSettings: state[`plugins-${pluginId}`].config,
    };
}

export default connect(mapStateToProps)(StillRoom);