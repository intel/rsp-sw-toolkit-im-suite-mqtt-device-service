rrpBuildGoCode {
    projectKey = 'gateway-device-service'
    testDependencies = ['mongo']
    dockerBuildOptions = ['--squash', '--build-arg GIT_COMMIT=$GIT_COMMIT']
    ecrRegistry = "280211473891.dkr.ecr.us-west-2.amazonaws.com"
    dockerImageName = "rsp/${projectKey}"
    protexProjectName = 'bb-gateway-device-service'

    infra = [
        stackName: 'RSP-Codepipeline-GatewayDeviceService'
    ]

    customBuildScript = "./build.sh"

    notify = [
        slack: [ success: '#ima-build-success', failure: '#ima-build-failed' ]
    ]
}
