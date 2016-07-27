'use strict';

angular.module('myApp.settings', ['ngRoute'])

.config(['$routeProvider', function($routeProvider) {
  $routeProvider.when('/settings', {
    templateUrl: '/views/settings/settings.html',
    controller: 'SettingsController'
  });
}])

.controller('SettingsController', function ($scope, $http) {
  $scope.getconfigData = function() {
    $http({
      method: 'GET',
      url: '/config/all'
    }).
    success(function (data, status, headers, config) {
      $scope.config = data;
    }).
    error(function (data, status, headers, config) {
      console.log('error');
    });
  };

  $scope.getconfigData();

});