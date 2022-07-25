// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

import React from 'react';
import PropTypes from 'prop-types';
import {FormattedMessage} from 'react-intl';
import {makeStyleFromTheme} from 'mattermost-redux/utils/theme_utils';

import {Svgs} from '../../constants';

export default class ChannelHeaderIcon extends React.PureComponent {
    render() {
        const style = getStyle();
        return (
            <span
                style={style.iconStyle}
                className='icon'
                aria-hidden='true'
                dangerouslySetInnerHTML={{__html: Svgs.VIDEO_CAMERA}}
            />
        );
    }
}

const getStyle = makeStyleFromTheme(() => {
    return {
        iconStyle: {
            position: 'relative',
            top: '-1px',
        },
    };
});
