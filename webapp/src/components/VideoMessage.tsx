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

import {createLocalVideoTrack, LocalVideoTrack, createLocalTracks} from 'livekit-client';

import {
    Flex,
    Grid,
    HStack,
    VStack,
    Box,
    Text,
    Icon,
    Button,
    ChakraProvider,
} from '@chakra-ui/react';

import {fetchToken} from '../actions';

// this is our demo server for demonstration purposes. It's easy to deploy your own.
// eslint-disable-next-line no-restricted-globals

const WSS_HOST = 'wss://livekit.k8s-local.cdek.ru';

// const WSS_HOST = '172.16.41.103:32766';
const RoomView = (props: any) => {
    console.log(props);
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
            <Box
                maxW='sm'
                borderWidth='1px'
                borderRadius='lg'
                overflow='hidden'
            >
                <Button
                    colorScheme='blue'
                    onClick={handleClick}
                >Подключиться</Button>
            </Box> :
            (
                <Box
                    w='100%'
                    h='100%'
                >
                    <LiveKitRoom
                        url={WSS_HOST}
                        token={token}
                        stageRenderer={StageView}
                        onConnected={(room) => {
                            handleConnected(room);
                        }}
                    />
                </Box>
            )}
    </>);
};

const CustomParticipantView = ({participant}) => {
    const {cameraPublication, isLocal} = useParticipant(participant);
    if (!cameraPublication || !cameraPublication.isSubscribed || !cameraPublication.track || cameraPublication.isMuted) {
        return null;
    }
    return (
        <Box
            w='95%'
            pos='relative'
            left='50%'
            transform='translateX(-50%)'
        >
            <VideoRenderer
                track={cameraPublication.track}
                isLocal={isLocal}
                objectFit='contain'
                width='100%'
                height='100%'
            />
        </Box>
    );
};

const RoomStatusView = ({children}) => (
    <VStack
        w='100%'
        h='100%'
        align='center'
        justify='center'
    >
        <Text
            textStyle='v2.h5-mono'
            color='#000'
            textTransform='uppercase'
            letterSpacing='0.05em'
        >{children}</Text>
    </VStack>
);

// renderStage prepares the layout of the stage using subcomponents. Feel free to
// modify as you see fit. It uses the built-in ParticipantView component in this
// example; you may use a custom component instead.
function StageView({roomState}) {
    const {room, participants, audioTracks, isConnecting, error} = roomState;
    console.log({room, participants, audioTracks, isConnecting, error});
    const gridRef = React.useRef(null);
    const [gridTemplateColumns, setGridTemplateColumns] = React.useState('1fr');

    React.useEffect(() => {
        const gridEl = gridRef.current;
        if (!gridEl || participants.length === 0) {
            return;
        }

        const totalWidth = gridEl.clientWidth;
        const numCols = Math.ceil(Math.sqrt(participants.length));
        const colSize = Math.floor(totalWidth / numCols);
        setGridTemplateColumns(`repeat(${numCols}, minmax(50px, ${colSize}px))`);
    }, [participants]);

    if (isConnecting) {
        return <RoomStatusView>Connecting...</RoomStatusView>;
    }
    if (error) {
        return <RoomStatusView>Error: {error.message}</RoomStatusView>;
    }
    if (!room) {
        return <RoomStatusView>Room closed</RoomStatusView>;
    }

    return (
        <Flex
            direction='column'
            justify='center'
            h='100%'
            bg='black'
        >
            <Grid
                ref={gridRef}
                __css={{
                    display: 'grid',
                    aspectRatio: '1.77778',
                    overflow: 'hidden',
                    background: 'black',
                    alignItems: 'center',
                    justifyContent: 'center',
                    width: '100%',
                    gridTemplateColumns,
                }}
            >
                {audioTracks.map((track) => (
                    <AudioRenderer
                        track={track}
                        key={track.sid}
                    />
                ))}
                {participants.map((participant) => (
                    <CustomParticipantView
                        key={participant.sid}
                        participant={participant}
                        showOverlay={true}
                        aspectWidth={16}
                        aspectHeight={9}
                    />
                ))}
            </Grid>
            {/*<ControlsView room={room}/>*/}
        </Flex>
    );
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

