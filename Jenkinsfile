rrpBuildGoCode {
    projectKey = 'gateway-device-service'
    testDependencies = ['mongo']
    dockerBuildOptions = ['--squash', '--build-arg GIT_COMMIT=$GIT_COMMIT']
    ecrRegistry = "280211473891.dkr.ecr.us-west-2.amazonaws.com"
    dockerImageName = "rsp/${projectKey}"

    infra = [
        stackName: 'RSP-Codepipeline-GatewayDeviceService'
    ]

    staticCodeScanBranch = 'delhi'

    notify = [
        slack: [ success: '#ima-build-success', failure: '#ima-build-failed' ]
    ]
}
