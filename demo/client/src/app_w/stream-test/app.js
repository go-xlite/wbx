import { pathPrefix } from '../../../lib/site_core.js';
let video = null;
let isMuted = false;


function formatTime(seconds) {
    if (isNaN(seconds)) return '00:00';
    const mins = Math.floor(seconds / 60);
    const secs = Math.floor(seconds % 60);
    return `${mins.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`;
}

function addMessage(msg, type = 'info') {
    const messagesDiv = document.getElementById('messages');
    const msgDiv = document.createElement('div');
    msgDiv.className = 'message ' + type;
    const timestamp = new Date().toLocaleTimeString();
    msgDiv.innerHTML = `<strong>[${timestamp}]</strong> ${msg}`;
    messagesDiv.appendChild(msgDiv);
    messagesDiv.scrollTop = messagesDiv.scrollHeight;
}

function updateStats() {
    if (!video) return;
    
    document.getElementById('duration').textContent = formatTime(video.duration);
    document.getElementById('currentTime').textContent = formatTime(video.currentTime);
    
    if (video.buffered.length > 0) {
        const bufferedEnd = video.buffered.end(video.buffered.length - 1);
        const bufferedPercent = (bufferedEnd / video.duration * 100).toFixed(0);
        document.getElementById('buffered').textContent = bufferedPercent + '%';
    }
}

function playVideo() {
    if (video) {
        video.play().then(() => {
            addMessage('Video playback started', 'success');
        }).catch(err => {
            addMessage('Error playing video: ' + err.message, 'error');
        });
    }
}

function pauseVideo() {
    if (video) {
        video.pause();
        addMessage('Video paused', 'info');
    }
}

function rewindVideo() {
    if (video) {
        video.currentTime = Math.max(0, video.currentTime - 10);
        addMessage('Rewound 10 seconds', 'info');
    }
}

function forwardVideo() {
    if (video) {
        video.currentTime = Math.min(video.duration, video.currentTime + 10);
        addMessage('Fast-forwarded 10 seconds', 'info');
    }
}

function toggleMute() {
    if (video) {
        isMuted = !isMuted;
        video.muted = isMuted;
        addMessage(isMuted ? 'Video muted' : 'Video unmuted', 'info');
    }
}

// Initialize on page load
function init() {
    video = document.getElementById('videoPlayer');
    const videoSource = document.getElementById('videoSource');

  
    
    const protocol = window.location.protocol;
    const streamUrl = `${protocol}//${window.location.host}/s/${pathPrefix.split('/').pop()}/stream/sharko_video.mp4`;
    
    // Update the source and displayed endpoint
    videoSource.src = streamUrl;
    video.load();
    document.getElementById('streamEndpoint').textContent = streamUrl;
    
    // Bind button event listeners
    document.getElementById('playBtn').addEventListener('click', playVideo);
    document.getElementById('pauseBtn').addEventListener('click', pauseVideo);
    document.getElementById('rewindBtn').addEventListener('click', rewindVideo);
    document.getElementById('forwardBtn').addEventListener('click', forwardVideo);
    document.getElementById('muteBtn').addEventListener('click', toggleMute);
    
    // Video event listeners
    video.addEventListener('loadedmetadata', function() {
        addMessage('Video metadata loaded - Duration: ' + formatTime(video.duration), 'success');
        updateStats();
    });
    
    video.addEventListener('loadstart', function() {
        addMessage('Loading video...', 'info');
    });
    
    video.addEventListener('canplay', function() {
        addMessage('Video ready to play', 'success');
    });
    
    video.addEventListener('play', function() {
        addMessage('Playback started', 'success');
    });
    
    video.addEventListener('pause', function() {
        addMessage('Playback paused', 'info');
    });
    
    video.addEventListener('ended', function() {
        addMessage('Playback ended', 'info');
    });
    
    video.addEventListener('timeupdate', function() {
        updateStats();
    });
    
    video.addEventListener('progress', function() {
        updateStats();
    });
    
    video.addEventListener('error', function(e) {
        let errorMsg = 'Unknown error';
        if (video.error) {
            switch (video.error.code) {
                case 1:
                    errorMsg = 'Video loading aborted';
                    break;
                case 2:
                    errorMsg = 'Network error';
                    break;
                case 3:
                    errorMsg = 'Video decoding failed';
                    break;
                case 4:
                    errorMsg = 'Video format not supported';
                    break;
            }
        }
        addMessage('Video error: ' + errorMsg, 'error');
    });
    
    video.addEventListener('seeking', function() {
        addMessage('Seeking to ' + formatTime(video.currentTime), 'info');
    });
    
    video.addEventListener('seeked', function() {
        addMessage('Seek complete', 'success');
    });
    
    addMessage('Media Streaming Test Console loaded', 'info');
    updateStats();
}

// Check if DOM is already loaded or wait for it
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
} else {
    // DOM already loaded, execute immediately
    init();
}
