angular.module('myApp.webSocket', ['ngWebSocket'])
    .factory('webSocket', function($websocket) {
        // Open a WebSocket connection 
        var dataStream = $websocket('ws://localhost:9050/');

        var collection = [];

        dataStream.onMessage(function(message) {
            collection.push(JSON.parse(message.data));
        });

        var methods = {
            collection: collection,
            get: function() {
                dataStream.send(JSON.stringify({ action: 'get' }));
            }
        };

        return methods;

    })