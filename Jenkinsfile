@Library('conservify') _

conservifyProperties([
    pipelineTriggers([])
])

timestamps {
    node {
        stage ('git') {
            checkout scm
        }

        stage ('build') {
            withEnv(["PATH+GOLANG=${tool 'golang-amd64'}/bin"]) {
                sh "make clean ci"
            }
        }

        stage ('archive') {
            archiveArtifacts "build/*/fake-device"
        }
    }
}
