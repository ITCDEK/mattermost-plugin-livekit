// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

import React from 'react';
import PropTypes from 'prop-types';
import {FormattedMessage} from 'react-intl';
import {makeStyleFromTheme} from 'mattermost-redux/utils/theme_utils';

import {id as pluginId} from '../../manifest';
import {Svgs} from '../../constants';

export default class ChannelHeaderIcon extends React.PureComponent {
    render() {
        const style = getStyle();
        return (
            <img
                style={style.iconStyle}
                className='icon'
                aria-hidden='true'
                src={`/plugins/${pluginId}/assets/channel-icon.png`}
                // dangerouslySetInnerHTML={{__html: Svgs.VIDEO_CAMERA}}
            />
        );
    }
}

const getStyle = makeStyleFromTheme((theme) => {
    return {
        iconStyle: {
            height: '130%',
            position: 'relative',
            top: '-1px',
        },
    };
});
