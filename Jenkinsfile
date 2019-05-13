rrpBuildGoCode {
    projectKey = 'mqtt-device-service'
    testDependencies = ['mongo']
    dockerBuildOptions = ['--squash', '--build-arg GIT_COMMIT=$GIT_COMMIT']
    ecrRegistry = "280211473891.dkr.ecr.us-west-2.amazonaws.com"
    dockerImageName = "rsp/${projectKey}"
    protexProjectName = 'bb-mqtt-device-service'

    infra = [
        stackName: 'RSP-Codepipeline-MqttDeviceService'
    ]

    customBuildScript = "./build.sh"

    notify = [
        slack: [ success: '#ima-build-success', failure: '#ima-build-failed' ]
    ]
}
