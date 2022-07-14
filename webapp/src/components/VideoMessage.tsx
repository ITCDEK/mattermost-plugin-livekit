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
import {
    useParticipant,
    VideoRenderer,
    AudioRenderer,
    LiveKitRoom,
} from '@livekit/react-components';

import {connect} from 'react-redux';
import Card from 'react-bootstrap/Card';
import Button from 'react-bootstrap/Button';
import Container from 'react-bootstrap/Container';
import Row from 'react-bootstrap/Row';
import Col from 'react-bootstrap/Col';
import ToggleButton from 'react-bootstrap/ToggleButton';
import ToggleButtonGroup from 'react-bootstrap/ToggleButtonGroup';
import {createLocalVideoTrack, LocalVideoTrack, createLocalTracks} from 'livekit-client';

import {fetchToken} from '../actions';

import './style.scss';

const WSS_HOST = 'wss://livekit.k8s-local.cdek.ru';

const RoomView = (props: any) => {
    // console.log(props);
    const [token, setToken] = React.useState('');
    const handleClick = () => {
        props.onFetchToken(props.post.id);
        setTimeout(() => {
            // @ts-ignore
            console.log('setToken', window.__token);

            // @ts-ignore
            setToken(window.__token);
        }, 500);
    };
    return (<>
        {!token ?
            <Card>
                <Card.Body>
                    <Button
                        variant='primary'
                        onClick={handleClick}
                    >Подключиться</Button>
                </Card.Body>
            </Card> :
            (
                <Card>
                    <LiveKitRoom
                        url={WSS_HOST}
                        token={token}
                        stageRenderer={StageView}
                        onConnected={(room) => {
                            handleConnected(room);
                        }}
                    />
                </Card>
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
    const handleOff = () => {
        room.disconnect();
    };
    const onToggleMic = () => {
        const enabled = room.localParticipant.isMicrophoneEnabled;
        room.localParticipant.setMicrophoneEnabled(!enabled);
    };
    const onToggleVideo = () => {
        const enabled = room.localParticipant.isCameraEnabled;
        room.localParticipant.setCameraEnabled(!enabled);
    };
    const onToggleScreen = () => {
        const enabled = room.localParticipant.isScreenShareEnabled;
        room.localParticipant.setScreenShareEnabled(!enabled);
    };

    return (<Container fluid={true}>
        <Row className='justify-content-md-center mb-3'>
            <Col lg={12}>
                <Card>
                    <Card.Body>
                        <Button
                            variant={room.localParticipant.isMicrophoneEnabled ? 'primary' : 'primary-outline'}
                            className='mr-3'
                            onClick={onToggleMic}
                        >
                            <i className={`CompassIcon ${room.localParticipant.isMicrophoneEnabled ? 'icon-microphone' : 'icon-microphone-off'}`}/>
                            {room.localParticipant.isMicrophoneEnabled ? 'Звук включен' : 'Звук выключен'}
                        </Button>
                        <Button
                            variant={room.localParticipant.isCameraEnabled ? 'primary' : 'primary-outline'}
                            className='mr-3'
                            onClick={onToggleVideo}
                        >
                            <i className='CompassIcon icon-camera-outline '/>
                            {room.localParticipant.isCameraEnabled ? 'Видео включено' : 'Видео выключено'}
                        </Button>
                        <Button
                            variant={'primary'}
                            className='mr-3'
                            onClick={onToggleScreen}
                        >
                            <i className='CompassIcon icon-monitor '/>
                            {room.localParticipant.isScreenShareEnabled ? 'Прекратить показ' : 'Показать экран'}
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

export default connect(null, {onFetchToken: fetchToken})(RoomView);

