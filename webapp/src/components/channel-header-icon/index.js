// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

import {connect} from 'react-redux';
import {id as pluginId} from '../../manifest';

import {getServerVersion} from 'mattermost-redux/selectors/entities/general';
import {isMinimumServerVersion} from 'mattermost-redux/utils/helpers';

import ChannelHeaderIcon from './channel-header-icon';

function mapStateToProps(state, ownProps) {
    return {
        ...ownProps,
        tokens: state[`plugins-${pluginId}`].tokens,
        pluginSettings: state[`plugins-${pluginId}`].config,
        // useSVG: !isMinimumServerVersion(getServerVersion(state), 5, 24),
    };
}

export default connect(mapStateToProps)(ChannelHeaderIcon);
