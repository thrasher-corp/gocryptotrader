import React, { Component } from 'react';

const withFetching = url => WrappedComponent =>
  class WithFetching extends Component {
    constructor(props) {
      super(props);

      this.state = {
        data: null,
        isLoading: false,
        error: null
      };
    }

    async componentDidMount() {
      this.mounted = true;
      if (this.mounted) {
        this.setState(prevState => ({ ...prevState, isLoading: true }));
      }
      try {
        const response = await fetch(url);
        const data = await response.json();

        if (this.mounted) {
          this.setState(prevState => ({
            ...prevState,
            data,
            isLoading: false
          }));
        }
      } catch (error) {
        if (this.mounted) {
          this.setState(prevState => ({
            ...prevState,
            error,
            isLoading: false
          }));
        }
      }
    }

    componentWillUnmount() {
      this.mounted = false;
    }

    render() {
      return <WrappedComponent {...this.props} {...this.state} />;
    }
  };

export default withFetching;
