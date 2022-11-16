export function showError(message) {
	this.setState({
		status: { variant: 'danger', icon: 'exclamation-octagon', text: message },
	});
	this.toast.current.toast();
}
