/*
  Playground lets you build a real-time video room using LiveKit's React component.
  Feel free to cutomize the code below to your heart's content.
  Send this page to a friend or open it in a new browser tab to see multi-user conferencing in action.

  In scope:
    `token`: An access token that joins the room, valid for 2h.
    `React`: The React library object
    `LiveKitReact`: The LiveKit React client SDK
    `LiveKitClient`: The LiveKit JavaScript client SDK
    `ReactFeather`: Feather icons to make things snazzy
    `Chakra`: ChakraUI for React
  */
import * as React from 'react';
import '@livekit/react-components/dist/index.css';
import {
    DisplayContext,
    DisplayOptions,
    useParticipant,
    ControlsProps,
    VideoRenderer,
    AudioRenderer,
    LiveKitRoom,
} from '@livekit/react-components';

import {Room, RoomEvent, setLogLevel, VideoPresets, createLocalVideoTrack, LocalVideoTrack, createLocalTracks} from 'livekit-client';

import {connect, useSelector, useDispatch} from 'react-redux';
import {defineMessages, useIntl } from 'react-intl';
import Card from 'react-bootstrap/Card';
import Button from 'react-bootstrap/Button';
import Container from 'react-bootstrap/Container';
import Row from 'react-bootstrap/Row';
import Col from 'react-bootstrap/Col';
import ToggleButton from 'react-bootstrap/ToggleButton';
import ToggleButtonGroup from 'react-bootstrap/ToggleButtonGroup';
import {fetchToken, getTranslation, deletePost} from '../actions';
import {id as pluginId} from '../manifest';

import StillRoom from './StillRoom';

import './style.scss';

const RoomView = (props: any) => {
    const dispatch = useDispatch();
    const ttl = Math.abs((new Date() - new Date(props.post.create_at)) / (1000 * 60 *60));
    if (ttl > 12) {
        console.log(`liveKit post is ${Math.round(ttl)} hour(s) old, deleting...`);
        dispatch(deletePost(props.post.id));
    }
    console.log(props.post.props.room_capacity);
    console.log(props.post.props.room_host);
    const [displayOptions, setDisplayOptions] = React.useState<DisplayOptions>({stageLayout: 'grid', showStats: false});
    const updateOptions = (options: DisplayOptions) => setDisplayOptions({...displayOptions, ...options});
    return (<>
        {!props.liveRooms[props.post.id] ?
            <StillRoom
                post = {props.post}
                token = {props.tokens[props.post.id]}
                theme = {props.theme}>
            </StillRoom> :
            // <Card>
            //     <Card.Body>
            //         <Button
            //             variant='primary'
            //             onClick={goLive}
            //         >{ getTranslation("room.connect")}</Button>
            //     </Card.Body>
            // </Card> :
            (
                <DisplayContext.Provider value={displayOptions}>
                <div className="roomContainer">
                    <LiveKitRoom
                        // https://livekit-users.slack.com/archives/C01KVTJH6BX/p1653607763178469
                        // https://github.com/livekit/livekit-react/blob/master/example/src/RoomPage.tsx
                        url={`wss://${props.pluginSettings.Host}:${props.pluginSettings.Port}`}
                        token={props.tokens[props.post.id]}
                        // stageRenderer={StageView}
                        // controlRenderer={controlsRenderer}
                        onConnected={(room) => {
                            setLogLevel('debug');
                            handleConnected(room);
                            // onConnected(room, query);
                        }}
                        onLeave={() => dispatch({type: "GO_STILL", data: props.post.id})}
                    />
                </div>
                </DisplayContext.Provider>
            )}
    </>);
};

const CustomParticipantView = ({participant}) => {
    const {cameraPublication, isLocal, screenSharePublication} = useParticipant(participant);
    console.log(cameraPublication, isLocal, screenSharePublication);
    if (!cameraPublication || !cameraPublication.isSubscribed || !cameraPublication.track || cameraPublication.isMuted) {
        return null;
    }
    return (
        <Card>
            <VideoRenderer
                track={screenSharePublication ? screenSharePublication.track : cameraPublication.track}
                isLocal={isLocal}
                objectFit='contain'
                width='100%'
                height='100%'
            />
        </Card>
    );
};

const RoomStatusView = ({children}) => (
    <Card><Card.Body>{children}</Card.Body></Card>
);

// renderStage prepares the layout of the stage using subcomponents. Feel free to
// modify as you see fit. It uses the built-in ParticipantView component in this
// example; you may use a custom component instead.
function StageView({roomState}) {
    // const dispatch = useDispatch();
    const {room, participants, audioTracks, isConnecting, error} = roomState;

    // console.log({room, participants, audioTracks, isConnecting, error});

    if (isConnecting) {
        return <RoomStatusView>Подключение...</RoomStatusView>;
    }
    if (error) {
        return <RoomStatusView>Ошибка: {error.message}</RoomStatusView>;
    }
    if (!room) {
        return <RoomStatusView>Комната закрыта</RoomStatusView>;
    }

    // const data = [...participants, ...participants, ...participants, ...participants, ...participants];
    const data = participants;
    let xxlCount = 6;
    let xlCount = 6;
    let lgCount = 6;
    let mdCount = 12;

    // if (participants.length > 2) {
    if (data.length > 2) {
        xxlCount = 3;
        xlCount = 3;
        lgCount = 4;
        mdCount = 6;
    }

    return (<Container fluid={true}>
        <Row>
            {data.map((participant) => (
                <Col
                    key={participant.sid}
                    xxl={xxlCount}
                    xl={xlCount}
                    lg={lgCount}
                    md={mdCount}
                >
                    <CustomParticipantView
                        participant={participant}
                        showOverlay={true}
                        aspectWidth={16}
                        aspectHeight={9}
                    />
                </Col>
            ))}
        </Row>
        {
            audioTracks.map((track) => (
                <AudioRenderer
                    track={track}
                    key={track.sid}
                />
            ))
        }
    </Container>)
    ;
}

function controlsRenderer(props: ControlsProps): React.ReactElement | null {
    const handleOff = () => {
        props.room.disconnect();
        // dispatch({type: "GO_STILL", data: "pass post.id here"});
    };
    const onToggleMic = () => {
        const enabled = props.room.localParticipant.isMicrophoneEnabled;
        props.room.localParticipant.setMicrophoneEnabled(!enabled);
    };
    const onToggleVideo = () => {
        const enabled = props.room.localParticipant.isCameraEnabled;
        props.room.localParticipant.setCameraEnabled(!enabled);
    };
    const onToggleScreen = () => {
        const enabled = props.room.localParticipant.isScreenShareEnabled;
        props.room.localParticipant.setScreenShareEnabled(!enabled);
    };

    return (<Container fluid={true}>
        <Row className='justify-content-md-center mb-3'>
            <Col lg={12}>
                <Card>
                    <Card.Body>
                        <Button
                            variant={props.room.localParticipant.isMicrophoneEnabled ? 'primary' : 'primary-outline'}
                            className='mr-3'
                            onClick={onToggleMic}
                        >
                            <i className={`CompassIcon ${props.room.localParticipant.isMicrophoneEnabled ? 'icon-microphone' : 'icon-microphone-off'}`}/>
                            {props.room.localParticipant.isMicrophoneEnabled ? 'Звук включен' : 'Звук выключен'}
                        </Button>
                        <Button
                            variant={props.room.localParticipant.isCameraEnabled ? 'primary' : 'primary-outline'}
                            className='mr-3'
                            onClick={onToggleVideo}
                        >
                            <i className='CompassIcon icon-camera-outline '/>
                            {props.room.localParticipant.isCameraEnabled ? 'Видео включено' : 'Видео выключено'}
                        </Button>
                        <Button
                            variant={'primary'}
                            className='mr-3'
                            onClick={onToggleScreen}
                        >
                            <i className='CompassIcon icon-monitor '/>
                            {props.room.localParticipant.isScreenShareEnabled ? 'Прекратить показ' : 'Показать экран'}
                        </Button>
                        <Button
                            variant='danger'
                            onClick={handleOff}
                        >
                            <i className='CompassIcon icon-phone-hangup '/>
                            Отключиться
                        </Button>
                    </Card.Body>
                </Card>
            </Col>
        </Row>
    </Container>)
    ;
}

async function handleConnected(room) {
    console.log('connected to room', room);

    const tracks = await createLocalTracks({
        audio: true,
        video: true,
    });
    tracks.forEach((track) => {
        room.localParticipant.publishTrack(track, {simulcast: true});
    });
}

function mapStateToProps(state, ownProps) {
    return {
        ...ownProps,
        tokens: state[`plugins-${pluginId}`].tokens,
        liveRooms: state[`plugins-${pluginId}`].liveRooms,
        pluginSettings: state[`plugins-${pluginId}`].config,
        // currentLocale: getCurrentUserLocale(state),
        // useSVG: !isMinimumServerVersion(getServerVersion(state), 5, 24),
    };
}

export default connect(mapStateToProps, {onFetchToken: fetchToken})(RoomView);

