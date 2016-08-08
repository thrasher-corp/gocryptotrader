'use strict';

// Declare app level module which depends on views, and components
angular.module('myApp', [
  'ngRoute',
  'ui-notification',
  'myApp.home',
  'myApp.about',
  'myApp.settings',
  'myApp.version'
]).
config(['$locationProvider', '$routeProvider' ,'NotificationProvider',  function($locationProvider, $routeProvider, NotificationProvider) {
  NotificationProvider.setOptions({
            delay: 10000,
            startTop: 60,
            startRight: 10,
            verticalSpacing: 10,
            horizontalSpacing: 20,
            positionX: 'right',
            positionY: 'top'
        });
  
  $locationProvider.hashPrefix('!');

  $routeProvider.otherwise({redirectTo: '/'});
}]);
