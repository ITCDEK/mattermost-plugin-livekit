import React from 'react';
import {connect} from 'react-redux';
import {getTheme} from 'mattermost-redux/selectors/entities/preferences';
import {makeStyleFromTheme} from 'mattermost-redux/utils/theme_utils';

import {getTranslation} from '../actions';
import {id as pluginId} from '../manifest';

const StillPost = (props) => {
    const buttonLabel = getTranslation("room.connect");
    const style = getStyle(props.theme);
    return (
        <div style={style.wrapper}>
            <div style={style.message}>{props.post.message}</div>
            <div style={style.buttonWrapper}>
                <div style={style.connectButton} className = "btn btn-lg btn-primary">{buttonLabel}</div>
            </div>
        </div>
    );
}

const getStyle = makeStyleFromTheme((theme) => {
    console.log(theme);
    return {
        wrapper: {
            width: "100%",
            display: "flex"
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
        },
        connectButton: {
            // color: theme.buttonColor,
        },
        button: {
            // color: theme.buttonColor,
            position: 'relative',
            top: '-1px',
        },
    };
});

function mapStateToProps(state, ownProps) {
    return {
        ...ownProps,
        // theme: getTheme(),
        tokens: state[`plugins-${pluginId}`].tokens,
        pluginSettings: state[`plugins-${pluginId}`].config,
    };
}

export default connect(mapStateToProps)(StillPost);