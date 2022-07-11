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
} from '@chakra-ui/react';

import {fetchToken} from '../actions';

// this is our demo server for demonstration purposes. It's easy to deploy your own.
// eslint-disable-next-line no-restricted-globals
const OBJ = location.search === '?sec' ? {
    token: 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ2aWRlbyI6eyJyb29tSm9pbiI6dHJ1ZSwicm9vbSI6ImJhc2UiLCJjYW5QdWJsaXNoIjp0cnVlLCJjYW5TdWJzY3JpYmUiOnRydWV9LCJpYXQiOjE2NTcyMDQ4MzMsIm5iZiI6MTY1NzIwNDgzMywiZXhwIjoxNjU3MjEyMDMzLCJpc3MiOiJBUElreldoYnhCYUdTaXEiLCJzdWIiOiJiYXNlNSIsImp0aSI6ImJhc2U1In0.ewn7ZXepPqAsUZ28edLKqYYfSEG7LizkViUQhnhN-i8',
    url: 'wss://demo.livekit.cloud',
} : {
    token: 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ2aWRlbyI6eyJyb29tSm9pbiI6dHJ1ZSwicm9vbSI6InJ0eSIsImNhblB1Ymxpc2giOnRydWUsImNhblN1YnNjcmliZSI6dHJ1ZX0sImlhdCI6MTY1NzIwNjMwMiwibmJmIjoxNjU3MjA2MzAyLCJleHAiOjE2NTcyMTM1MDIsImlzcyI6IkFQSWt6V2hieEJhR1NpcSIsInN1YiI6InJ0eTEiLCJqdGkiOiJydHkxIn0.QEMAJgz9QgHA0C77I7UzYra8vkT5qngNrfiJ2F_c8ms',
    url: 'wss://demo.livekit.cloud',
};
const RoomView = (props:any) => {
    console.log(props);
    const [token, setToken] = React.useState('');
    const onClick = () => {
        props.onFetchToken(props.post.id);
    };
    return !token ? <div style={{height: '50px', width: '150px'}}>
        <button onClick={onClick}>GET TOKEN</button>
    </div> : (
        <Box
            w='100%'
            h='100%'
        >
            <LiveKitRoom
                url={OBJ.url}
                token={OBJ.token}
                stageRenderer={StageView}
                onConnected={(room) => {
                    handleConnected(room);
                }}
            />
        </Box>
    );
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
